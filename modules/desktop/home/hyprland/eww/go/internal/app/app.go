// Package app wires the daemon together: seed the state, start the control
// server, launch the watchers, and emit.
package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ewwbar/internal/collect"
	"ewwbar/internal/control"
	"ewwbar/internal/paths"
	"ewwbar/internal/state"
	"ewwbar/internal/watch"
)

// cpuSampleGap is the window collectors.cpu_state measures over: two reads of
// /proc/stat with a pause between them, because the file holds totals rather
// than rates.
const cpuSampleGap = 200 * time.Millisecond

// Run is app.run_bar.
func Run() int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// SIGPIPE must be ignored, not fatal. eww closes stdout when it stops
	// listening, and Go's runtime kills the process on a SIGPIPE from fd 1 or 2
	// -- where CPython's signal.SIG_IGN leaves the write to fail as EPIPE and
	// lets EmitLoop return cleanly. Notify without reading is how you ignore.
	signal.Notify(make(chan os.Signal, 1), syscall.SIGPIPE)

	// SIGUSR1 asks the idle watcher to re-read now. Buffered and dropped when
	// full: the signal means "something changed", and two of them mean the same
	// as one.
	idleRefresh := make(chan struct{}, 1)
	usr1 := make(chan os.Signal, 1)
	signal.Notify(usr1, syscall.SIGUSR1)
	go func() {
		for range usr1 {
			select {
			case idleRefresh <- struct{}{}:
			default:
			}
		}
	}()

	stopping := make(chan os.Signal, 1)
	signal.Notify(stopping, syscall.SIGTERM, syscall.SIGINT)

	removePidfile := control.WriteBackendPidfile()
	defer removePidfile()

	store := state.New()

	listener, err := control.Listen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "eww-bar control server: %v\n", err)
	} else {
		defer listener.Close()
		go control.Serve(store, listener)
	}

	seed(store)
	launchWatchers(ctx, store, idleRefresh)

	go func() {
		<-stopping
		cancel()
	}()

	watch.EmitLoop(ctx, store, os.Stdout)
	return 0
}

// seed fills the state once before any watcher runs, so the bar renders real
// values immediately rather than the defaults for the first few seconds.
func seed(store *state.Store) {
	activeWindow := collect.CollectActiveWindow()
	clock := collect.ClockState()
	media := collect.CollectMedia()
	cpu := collect.CollectCPU(func() { time.Sleep(cpuSampleGap) })
	memory := collect.CollectMemory()
	temperature := collect.CollectTemperature()
	network := collect.CollectNetwork()
	volume := collect.CollectVolume(true)
	battery := collect.CollectBattery()
	bluetooth := collect.CollectBluetooth()
	idle := collect.IdleInhibitedState()
	display := collect.DisplayState("")
	workspace, monitorWorkspaces := collect.CollectWorkspaceViews()
	notifications := collect.CollectNotifications()

	store.Update(func(bar *state.Bar) {
		bar.ActiveWindow = activeWindow
		bar.Clock = clock
		bar.Media = media
		bar.CPU = cpu
		bar.Memory = memory
		bar.Temperature = temperature
		bar.Network = network
		bar.Volume = volume
		bar.Battery = battery
		bar.Bluetooth = bluetooth
		bar.IdleInhibited = idle
		bar.Display = display
		bar.WorkspaceState = workspace
		bar.Notifications = notifications
	})

	// After the store, not before: this spawns `eww update`, and the seed's job
	// is to get the first snapshot onto stdout quickly. The bar renders without
	// the variable anyway -- its default is the zero value, which the yuck reads
	// as "one screen" and falls back to bar_state.workspace_state.
	watch.PublishMonitorWorkspaces(monitorWorkspaces)
}

// launchWatchers starts every background loop.
//
// The intervals are the original's and each is a deliberate cost trade rather
// than a round number: cpu and memory at 7 s because they are two file reads,
// temperature at 15 s, battery and notifications at 30 s, network and clock at
// 60 s, tray at 5 s because it is one cheap busctl call.
func launchWatchers(ctx context.Context, store *state.Store, idleRefresh <-chan struct{}) {
	go watch.WatchHyprland(ctx, store)
	go watch.WatchIdleInhibitor(ctx, store, idleRefresh)

	go watch.Periodic(ctx, store, 60*time.Second, func() watch.Update {
		clock := collect.ClockState()
		return func(bar *state.Bar) { bar.Clock = clock }
	})

	// Restore last night's quota cards before anything can probe for new ones,
	// so the popup opens on real numbers relabelled "as of <time>" rather than
	// on "Waiting for first update...". Seeding only fills an empty cache, so
	// this cannot overwrite a probe that has already landed.
	if restored, ok := collect.LoadQuotaSnapshot(paths.AiQuotas()); ok {
		collect.SeedQuotaCache(restored)
	}

	// The fast path, and the reason it is separate from the loop below.
	//
	// AiUsageState computes the ccusage report and the quota probe and publishes
	// once, at the end. ccusage drives the bar face and the probe drives only
	// the popup's cards, so the bar used to sit at "-- " for as long as the
	// probe took while the numbers for it sat finished in a local variable.
	//
	// How long that is varies a lot -- 6.4 s measured end to end here against
	// 0.21 s for the ccusage half, and 41.6 s for the probe alone in the
	// measurement recorded at QuotaStates. The point is not the multiple, which
	// moves with the provider's latency, but that the bar face no longer
	// depends on it at all.
	//
	// Guarded on the state still being the placeholder: if the cycle happens to
	// win the race, its answer is strictly better and must not be overwritten.
	go func() {
		fast := collect.AiUsageFast()
		if fast.Source == "missing" {
			return
		}
		store.Update(func(bar *state.Bar) {
			if bar.AiUsage.Source == "missing" {
				bar.AiUsage = fast
			}
		})
	}()

	// AiRefreshCycle, not RefreshAiUsage directly: the ccusage report behind
	// the bar label runs every pass, but the openusage probe behind the
	// popup's quota cards is fifty times more expensive and only runs every
	// sixth. The popup refreshes itself on open.
	cycle := collect.AiRefreshCycle(6)
	go watch.PeriodicRefresh(ctx, store, 300*time.Second, func(s *state.Store) {
		fresh := cycle(s.Get().AiUsage, func(value collect.AiUsage) {
			s.Update(func(bar *state.Bar) { bar.AiUsage = value })
		})
		// Persisted here rather than inside QuotaStates because the collector
		// has no business knowing a path, and because SaveQuotaSnapshot drops
		// anything short of a fully live set on its own.
		_ = collect.SaveQuotaSnapshot(paths.AiQuotas(), fresh.Quotas, 0)
	})

	// The history grid on its own timer, at ten minutes.
	//
	// Not folded into the AI cycle above: that one exists to keep the bar face
	// current, while this reads a year of daily rollups that change once a day
	// and pushes a 22 KB variable to eww. Tying them would either make the grid
	// republish every five minutes for nothing, or slow the bar down to the
	// grid's cadence.
	go watch.Periodic(ctx, store, 600*time.Second, func() watch.Update {
		watch.PublishAiHistory(collect.AiHistoryState(paths.AiHistory()))
		return nil
	})

	go watch.Periodic(ctx, store, 7*time.Second, func() watch.Update {
		cpu := collect.CollectCPU(func() { time.Sleep(cpuSampleGap) })
		memory := collect.CollectMemory()
		return func(bar *state.Bar) { bar.CPU = cpu; bar.Memory = memory }
	})

	go watch.Periodic(ctx, store, 15*time.Second, func() watch.Update {
		temperature := collect.CollectTemperature()
		return func(bar *state.Bar) { bar.Temperature = temperature }
	})

	go watch.Periodic(ctx, store, 30*time.Second, func() watch.Update {
		battery := collect.CollectBattery()
		notifications := collect.CollectNotifications()
		return func(bar *state.Bar) {
			bar.Battery = battery
			bar.Notifications = notifications
		}
	})

	go watch.Periodic(ctx, store, 60*time.Second, func() watch.Update {
		network := collect.CollectNetwork()
		return func(bar *state.Bar) { bar.Network = network }
	})

	go watch.Periodic(ctx, store, 5*time.Second, func() watch.Update {
		count := collect.CollectTrayCount()
		return func(bar *state.Bar) { bar.TrayCount = count }
	})

	go watch.WatchCommand(ctx, store, nil, func() watch.Update {
		network := collect.CollectNetwork()
		return func(bar *state.Bar) { bar.Network = network }
	}, "nmcli", "monitor")

	// The line filter is load-bearing, not an optimisation. volume_state shells
	// out to the same subsystem pactl is reporting on, so without it the
	// watcher becomes its own event source -- measured at 1248 events a minute,
	// against 82 with the filter.
	go watch.WatchCommand(ctx, store, collect.VolumeEventIsRelevant, func() watch.Update {
		volume := collect.CollectVolume(false)
		return func(bar *state.Bar) { bar.Volume = volume }
	}, "pactl", "subscribe")

	go watch.WatchCommand(ctx, store, nil, func() watch.Update {
		media := collect.CollectMedia()
		return func(bar *state.Bar) { bar.Media = media }
	}, "playerctl", "-F", "metadata", "--format",
		"{{playerName}} {{status}} {{artist}} {{title}}")

	go watch.WatchCommand(ctx, store, nil, func() watch.Update {
		notifications := collect.CollectNotifications()
		return func(bar *state.Bar) { bar.Notifications = notifications }
	}, "dbus-monitor", "--profile", "interface='org.freedesktop.Notifications'")

	go watch.WatchBluetooth(ctx, store)

	// One shot, off the startup path: scanning the wallpaper directory builds a
	// thumbnail per image on a cold cache, which must not delay the first emit.
	go func() {
		wallpaper := collect.CollectWallpaper()
		store.Update(func(bar *state.Bar) { bar.Wallpaper = wallpaper })
	}()
}
