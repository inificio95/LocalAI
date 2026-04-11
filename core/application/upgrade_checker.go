package application

import (
	"context"
	"sync"
	"time"

	"github.com/mudler/LocalAI/core/config"
	"github.com/mudler/LocalAI/core/gallery"
	"github.com/mudler/LocalAI/pkg/model"
	"github.com/mudler/LocalAI/pkg/system"
	"github.com/mudler/xlog"
)

// UpgradeChecker periodically checks for backend upgrades and optionally
// auto-upgrades them. It caches the last check results for API queries.
type UpgradeChecker struct {
	appConfig   *config.ApplicationConfig
	modelLoader *model.ModelLoader
	galleries   []config.Gallery
	systemState *system.SystemState

	checkInterval time.Duration
	stop          chan struct{}
	done          chan struct{}
	triggerCh     chan struct{}

	mu            sync.RWMutex
	lastUpgrades  map[string]gallery.UpgradeInfo
	lastCheckTime time.Time
}

// NewUpgradeChecker creates a new UpgradeChecker service.
func NewUpgradeChecker(appConfig *config.ApplicationConfig, ml *model.ModelLoader) *UpgradeChecker {
	return &UpgradeChecker{
		appConfig:     appConfig,
		modelLoader:   ml,
		galleries:     appConfig.BackendGalleries,
		systemState:   appConfig.SystemState,
		checkInterval: 6 * time.Hour,
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
		triggerCh:     make(chan struct{}, 1),
		lastUpgrades:  make(map[string]gallery.UpgradeInfo),
	}
}

// Run starts the upgrade checker loop. It waits 30 seconds after startup,
// performs an initial check, then re-checks every 6 hours.
func (uc *UpgradeChecker) Run(ctx context.Context) {
	defer close(uc.done)

	// Initial delay: don't slow down startup
	select {
	case <-ctx.Done():
		return
	case <-uc.stop:
		return
	case <-time.After(30 * time.Second):
	}

	// First check
	uc.runCheck(ctx)

	// Periodic loop
	ticker := time.NewTicker(uc.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-uc.stop:
			return
		case <-ticker.C:
			uc.runCheck(ctx)
		case <-uc.triggerCh:
			uc.runCheck(ctx)
		}
	}
}

// Shutdown stops the upgrade checker loop.
func (uc *UpgradeChecker) Shutdown() {
	close(uc.stop)
	<-uc.done
}

// TriggerCheck forces an immediate upgrade check.
func (uc *UpgradeChecker) TriggerCheck() {
	select {
	case uc.triggerCh <- struct{}{}:
	default:
		// Already triggered, skip
	}
}

// GetAvailableUpgrades returns the cached upgrade check results.
func (uc *UpgradeChecker) GetAvailableUpgrades() map[string]gallery.UpgradeInfo {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	// Return a copy to avoid races
	result := make(map[string]gallery.UpgradeInfo, len(uc.lastUpgrades))
	for k, v := range uc.lastUpgrades {
		result[k] = v
	}
	return result
}

func (uc *UpgradeChecker) runCheck(ctx context.Context) {
	upgrades, err := gallery.CheckBackendUpgrades(ctx, uc.galleries, uc.systemState)

	uc.mu.Lock()
	uc.lastCheckTime = time.Now()
	if err != nil {
		xlog.Debug("Backend upgrade check failed", "error", err)
		uc.mu.Unlock()
		return
	}
	uc.lastUpgrades = upgrades
	uc.mu.Unlock()

	if len(upgrades) == 0 {
		xlog.Debug("All backends up to date")
		return
	}

	// Log available upgrades
	for name, info := range upgrades {
		if info.AvailableVersion != "" {
			xlog.Info("Backend upgrade available",
				"backend", name,
				"installed", info.InstalledVersion,
				"available", info.AvailableVersion)
		} else {
			xlog.Info("Backend upgrade available (new build)",
				"backend", name)
		}
	}

	// Auto-upgrade if enabled
	if uc.appConfig.AutoUpgradeBackends {
		for name, info := range upgrades {
			xlog.Info("Auto-upgrading backend", "backend", name,
				"from", info.InstalledVersion, "to", info.AvailableVersion)
			if err := gallery.UpgradeBackend(ctx, uc.systemState, uc.modelLoader,
				uc.galleries, name, nil); err != nil {
				xlog.Error("Failed to auto-upgrade backend",
					"backend", name, "error", err)
			} else {
				xlog.Info("Backend upgraded successfully", "backend", name,
					"version", info.AvailableVersion)
			}
		}
		// Re-check to update cache after upgrades
		if freshUpgrades, err := gallery.CheckBackendUpgrades(ctx, uc.galleries, uc.systemState); err == nil {
			uc.mu.Lock()
			uc.lastUpgrades = freshUpgrades
			uc.mu.Unlock()
		}
	}
}
