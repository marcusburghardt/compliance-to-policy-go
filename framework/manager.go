/*
 Copyright 2024 The OSCAL Compass Authors
 SPDX-License-Identifier: Apache-2.0
*/

package framework

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/oscal-compass/oscal-sdk-go/rules"
	"github.com/oscal-compass/oscal-sdk-go/settings"

	"github.com/oscal-compass/compliance-to-policy-go/v2/framework/config"
	"github.com/oscal-compass/compliance-to-policy-go/v2/plugin"
	"github.com/oscal-compass/compliance-to-policy-go/v2/policy"
)

// PluginManager manages the plugin lifecycle and compliance-to-policy
// workflows.
type PluginManager struct {
	// pluginDir is the location to search for plugins.
	pluginDir string
	// rulesStore contains indexed information about available RuleSets
	// which can be searched for the component title.
	rulesStore rules.Store
	// pluginIdMap stores resolved plugin IDs to the original component title from the
	// OSCAL Component Definitions.
	//
	// The original component title is needed to get information for the validation
	// component in the rules.Store (which provides input for the corresponding policy.Provider
	// plugin).
	pluginIdMap map[string]string
	// clientFactory is the function used to
	// create new plugin clients.
	clientFactory plugin.ClientFactoryFunc
	// logger for the PluginManager
	log hclog.Logger
}

// NewPluginManager creates a new instance of a PluginManager from a C2PConfig that can be used to
// interact with supported plugins.
//
// It supports the plugin lifecycle with the following methods:
//   - Finding and initializing plugins: LaunchPolicyPlugins()
//   - Execution - GeneratePolicy() and AggregateResults()
//   - Clean/Stop - Clean()
func NewPluginManager(cfg *config.C2PConfig) (*PluginManager, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Resolve all the options that were set in the C2PConfig into loaded structures
	// that are immediately usable for the PluginManager.
	rulesStore, pluginIDMap, err := config.ResolveOptions(cfg)
	if err != nil {
		return nil, err
	}

	return &PluginManager{
		pluginDir:     cfg.PluginDir,
		rulesStore:    rulesStore,
		clientFactory: plugin.ClientFactory(cfg.Logger),
		pluginIdMap:   pluginIDMap,
		log:           cfg.Logger,
	}, nil
}

// LaunchPolicyPlugins retrieves information for the plugins that have been requested
// in the C2PConfig and launches each plugin to make it ready for use with GeneratePolicy() and
// AggregateResults().
func (m *PluginManager) LaunchPolicyPlugins() (map[string]policy.Provider, error) {
	providerIds := make([]string, 0, len(m.pluginIdMap))
	for id := range m.pluginIdMap {
		providerIds = append(providerIds, id)
	}
	pluginsByIds := make(map[string]policy.Provider)

	m.log.Info(fmt.Sprintf("Searching for plugins in %s", m.pluginDir))

	pluginManifests, err := plugin.FindPlugins(
		m.pluginDir,
		plugin.WithProviderIds(providerIds),
		plugin.WithPluginType(plugin.PVPPluginName),
	)
	if err != nil {
		return pluginsByIds, err
	}
	m.log.Debug(fmt.Sprintf("Found %d matching plugins", len(pluginManifests)))

	for _, manifest := range pluginManifests {
		policyPlugin, err := plugin.NewPolicyPlugin(manifest, m.clientFactory)
		if err != nil {
			return pluginsByIds, err
		}
		pluginsByIds[manifest.ID] = policyPlugin
		m.log.Debug(fmt.Sprintf("Launching plugin %s", manifest.ID))
	}
	return pluginsByIds, nil
}

// GeneratePolicy identifies policy configuration for each provider in the given pluginSet to execute the Generate() method
// each policy.Provider. The rule set passed to each plugin can be configured with compliance specific settings with the
// complianceSettings input.
func (m *PluginManager) GeneratePolicy(ctx context.Context, pluginSet map[string]policy.Provider, complianceSettings settings.Settings) error {
	for providerId, policyPlugin := range pluginSet {
		componentTitle, ok := m.pluginIdMap[providerId]
		if !ok {
			m.log.Warn(fmt.Sprintf("skipping %s provider: missing validation component", providerId))
			continue
		}
		m.log.Debug(fmt.Sprintf("Generating policy for provider %s", providerId))

		appliedRuleSet, err := settings.ApplyToComponent(ctx, componentTitle, m.rulesStore, complianceSettings)
		if err != nil {
			return fmt.Errorf("failed to get rule sets for component %s: %w", componentTitle, err)
		}
		if err := policyPlugin.Generate(appliedRuleSet); err != nil {
			return fmt.Errorf("plugin %s: %w", providerId, err)
		}
	}
	return nil
}

// AggregateResults identifies policy configuration for each provider in the given pluginSet to execute the GetResults() method
// each policy.Provider. The rule set passed to each plugin can be configured with compliance specific settings with the
// complianceSettings input.
func (m *PluginManager) AggregateResults(ctx context.Context, pluginSet map[string]policy.Provider, complianceSettings settings.Settings) ([]policy.PVPResult, error) {
	var allResults []policy.PVPResult
	for providerId, policyPlugin := range pluginSet {
		// get the provider ids here to grab the policy
		componentTitle, ok := m.pluginIdMap[providerId]
		if !ok {
			return allResults, fmt.Errorf("missing title for provider %s", providerId)
		}
		m.log.Debug(fmt.Sprintf("Aggregating results for provider %s", providerId))
		appliedRuleSet, err := settings.ApplyToComponent(ctx, componentTitle, m.rulesStore, complianceSettings)
		if err != nil {
			return allResults, fmt.Errorf("failed to get rule sets for component %s: %w", componentTitle, err)
		}

		pluginResults, err := policyPlugin.GetResults(appliedRuleSet)
		if err != nil {
			return allResults, fmt.Errorf("plugin %s: %w", providerId, err)
		}
		allResults = append(allResults, pluginResults)
	}
	return allResults, nil
}

// Clean deletes managed instances of plugin clients that have been created using LaunchPolicyPlugins.
// This will remove all clients launched with the plugin.ClientFactoryFunc.
func (m *PluginManager) Clean() {
	m.log.Debug("Cleaning launched plugins")
	plugin.Cleanup()
}
