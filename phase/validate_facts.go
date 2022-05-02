package phase

import (
	"fmt"

	"github.com/k0sproject/version"
	log "github.com/sirupsen/logrus"
)

// ValidateFacts performs remote OS detection
type ValidateFacts struct {
	GenericPhase
	SkipDowngradeCheck bool
}

// Title for the phase
func (p *ValidateFacts) Title() string {
	return "Validate facts"
}

func (p *ValidateFacts) Initialize(g Getter) error {
	if v, ok := g.Value("disable-downgrade-check").(bool); ok {
		p.SkipDowngradeCheck = v
	}
	return nil
}

// Run the phase
func (p *ValidateFacts) Run() error {
	if err := p.validateDowngrade(); err != nil {
		return err
	}

	if err := p.validateDefaultVersion(); err != nil {
		return err
	}

	return nil
}

func (p *ValidateFacts) validateDowngrade() error {
	if p.SkipDowngradeCheck {
		return nil
	}
	if p.Config.Spec.K0sLeader().Metadata.K0sRunningVersion == "" {
		return nil
	}

	cfgV, err := version.NewVersion(p.Config.Spec.K0s.Version)
	if err != nil {
		return err
	}

	runV, err := version.NewVersion(p.Config.Spec.K0sLeader().Metadata.K0sRunningVersion)
	if err != nil {
		return err
	}

	if runV.GreaterThan(cfgV) {
		return fmt.Errorf("can't perform a downgrade: %s > %s", runV.String(), cfgV.String())
	}

	return nil
}

func (p *ValidateFacts) validateDefaultVersion() error {
	// Only check when running with a defaulted version
	if !p.Config.Spec.K0s.Metadata.VersionDefaulted {
		return nil
	}

	// Installing a fresh latest is ok
	if p.Config.Spec.K0sLeader().Metadata.K0sRunningVersion == "" {
		return nil
	}

	cfgV, err := version.NewVersion(p.Config.Spec.K0s.Version)
	if err != nil {
		return err
	}

	runV, err := version.NewVersion(p.Config.Spec.K0sLeader().Metadata.K0sRunningVersion)
	if err != nil {
		return err
	}

	// Upgrading should not be performed if the config version was defaulted
	if cfgV.GreaterThan(runV) {
		log.Warnf("spec.k0s.version was automatically defaulted to %s but the cluster is running %s", p.Config.Spec.K0s.Version, runV.String())
		log.Warnf("to perform an upgrade, set the k0s version in the configuration explicitly")
		p.Config.Spec.K0s.Version = runV.String()
		for _, h := range p.Config.Spec.Hosts {
			h.Metadata.NeedsUpgrade = false
		}
	}

	return nil
}
