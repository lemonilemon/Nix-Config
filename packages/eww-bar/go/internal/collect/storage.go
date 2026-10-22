package collect

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const glyphStorage = "\U000F02CA" // U+F02CA harddisk

const (
	storagePath    = "/nix/store"
	nhCleanService = "nh-clean.service"
	nhCleanTimer   = "nh-clean.timer"
)

// FsUsage is one statfs reading, in bytes.
type FsUsage struct {
	Total uint64
	Free  uint64
	Avail uint64
	OK    bool
}

// StatFS is the statfs seam.
var StatFS = func(path string) FsUsage {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return FsUsage{}
	}
	block := uint64(st.Bsize)
	return FsUsage{Total: st.Blocks * block, Free: st.Bfree * block, Avail: st.Bavail * block, OK: true}
}

// StorageCleanup is the nh-clean timer as the storage popup shows it.
type StorageCleanup struct {
	Status string `json:"status"` // waiting | missing | idle | running | failed
	Last   string `json:"last"`
	Next   string `json:"next"`
}

type Storage struct {
	Text        string         `json:"text"`
	Tooltip     string         `json:"tooltip"`
	Class       string         `json:"class"`
	Free        string         `json:"free"`
	Used        string         `json:"used"`
	UsedPercent int            `json:"used_percent"`
	Cleanup     StorageCleanup `json:"cleanup"`
}

func StorageDefault() Storage {
	return Storage{
		Text: glyphStorage + " --", Tooltip: "", Class: "",
		Free: "--", Used: "", UsedPercent: 0,
		Cleanup: StorageCleanup{Status: "waiting", Last: "", Next: ""},
	}
}

// FormatSize is bytes in the bar's G/T notation, with M below a gigabyte.
func FormatSize(bytes float64) string {
	gib := bytes / (1 << 30)
	switch {
	case gib >= 1000:
		return fmt.Sprintf("%.1fT", gib/1024)
	case gib >= 10:
		return fmt.Sprintf("%.0fG", gib)
	case gib >= 1:
		return fmt.Sprintf("%.1fG", gib)
	}
	return fmt.Sprintf("%.0fM", bytes/(1<<20))
}

// unitShow is one unit's block of `systemctl show` output.
type unitShow map[string]string

// ParseUnitShow splits `systemctl show a b` into one map per unit, in argument
// order.
func ParseUnitShow(text string) []unitShow {
	var units []unitShow
	current := unitShow{}
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			if len(current) > 0 {
				units = append(units, current)
				current = unitShow{}
			}
			continue
		}
		if key, value, found := strings.Cut(line, "="); found {
			current[key] = value
		}
	}
	if len(current) > 0 {
		units = append(units, current)
	}
	return units
}

// unixStamp reads a --timestamp=unix value such as "@1792605600".
func unixStamp(value string) float64 {
	seconds, err := strconv.ParseFloat(strings.TrimPrefix(value, "@"), 64)
	if err != nil || seconds <= 0 {
		return 0
	}
	return seconds
}

// CleanupLastRun is when the cleanup last started: the timer's trigger, which
// survives a reboot, or the service's own start when it was started by hand.
func CleanupLastRun(units []unitShow) float64 {
	if len(units) < 2 {
		return 0
	}
	return math.Max(unixStamp(units[1]["LastTriggerUSec"]), unixStamp(units[0]["ExecMainStartTimestamp"]))
}

var (
	nixFreedPattern  = regexp.MustCompile(`store paths deleted, ([0-9.]+) ((?:[KMGT]i)?B) freed`)
	nixFreedUnitSize = map[string]float64{"B": 1, "KiB": 1 << 10, "MiB": 1 << 20, "GiB": 1 << 30, "TiB": 1 << 40}
)

// CleanupJournal is what the last run's journal says: the space nix reported
// freeing, and whether systemd recorded a failure.
func CleanupJournal(text string) (freed string, failed bool) {
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "Failed with result") {
			failed = true
		}
		if match := nixFreedPattern.FindStringSubmatch(line); match != nil {
			if amount, err := strconv.ParseFloat(match[1], 64); err == nil {
				freed = FormatSize(amount * nixFreedUnitSize[match[2]])
			}
		}
	}
	return freed, failed
}

// cleanupDay names a moment the way the popup says it: by calendar day, so a run
// at 23:50 is "yesterday" ten minutes later.
func cleanupDay(epoch, nowEpoch float64) string {
	at := localTime(epoch)
	now := localTime(nowOr(nowEpoch))
	atDay := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, at.Location())
	nowDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	days := int(math.Round(atDay.Sub(nowDay).Hours() / 24))

	switch {
	case days == 0:
		return "today " + at.Format("15:04")
	case days == -1:
		return "yesterday"
	case days == 1:
		return "tomorrow"
	case days < -1 && days > -7:
		return fmt.Sprintf("%d days ago", -days)
	case days > 1 && days < 7:
		return at.Format("Monday")
	}
	return at.Format("Jan 2")
}

// CleanupStateFromParts reads `systemctl show nh-clean.service nh-clean.timer`
// and the last run's journal.
func CleanupStateFromParts(showText, journalText string, nowEpoch float64) StorageCleanup {
	units := ParseUnitShow(showText)
	if len(units) < 2 || units[1]["LoadState"] != "loaded" {
		return StorageCleanup{Status: "missing", Last: "", Next: ""}
	}
	service, timer := units[0], units[1]
	freed, journalFailed := CleanupJournal(journalText)
	lastRun := CleanupLastRun(units)

	cleanup := StorageCleanup{Status: "idle"}
	switch {
	case service["ActiveState"] == "activating":
		cleanup.Status, cleanup.Last = "running", "Running now"
	case lastRun == 0:
		cleanup.Last = "Never"
	case journalFailed || (service["ExecMainStartTimestamp"] != "" && service["Result"] != "success"):
		cleanup.Status, cleanup.Last = "failed", "Failed "+cleanupDay(lastRun, nowEpoch)
	default:
		cleanup.Last = cleanupDay(lastRun, nowEpoch)
		if freed != "" {
			cleanup.Last += " · freed " + freed
		}
	}

	if next := unixStamp(timer["NextElapseUSecRealtime"]); timer["ActiveState"] == "active" && next > 0 {
		cleanup.Next = cleanupDay(next, nowEpoch)
	} else {
		cleanup.Next = "Timer stopped"
	}
	return cleanup
}

// StorageStateFromParts builds the module from a statfs reading and the cleanup
// state. Used is df's figure: of the space an unprivileged user could ever have,
// how much is taken, rounded up.
func StorageStateFromParts(fs FsUsage, cleanup StorageCleanup) Storage {
	if !fs.OK || fs.Total == 0 {
		state := StorageDefault()
		state.Cleanup = cleanup
		return state
	}
	used := float64(fs.Total - fs.Free)
	pct := 0
	if used+float64(fs.Avail) > 0 {
		pct = int(math.Ceil(used / (used + float64(fs.Avail)) * 100))
	}
	class := ""
	switch {
	case pct >= 95:
		class = "critical"
	case pct >= 90:
		class = "warning"
	}
	free := FormatSize(float64(fs.Avail))
	tooltip := fmt.Sprintf("Free: %s of %s (%d%% used)", free, FormatSize(float64(fs.Total)), pct)
	if cleanup.Status != "missing" && cleanup.Status != "waiting" {
		tooltip += "\nLast cleanup: " + cleanup.Last + "\nNext cleanup: " + cleanup.Next
	}
	return Storage{
		Text:        glyphStorage + " " + free,
		Tooltip:     tooltip,
		Class:       class,
		Free:        free,
		Used:        fmt.Sprintf("%s of %s", FormatSize(used), FormatSize(float64(fs.Total))),
		UsedPercent: pct,
		Cleanup:     cleanup,
	}
}

var (
	cleanupJournalLock sync.Mutex
	cleanupJournalRun  float64
	cleanupJournalText string
)

// cleanupJournalFor reads the journal from lastRun on, once per run. The slack
// covers the trigger stamp and the first log line landing in different seconds.
func cleanupJournalFor(lastRun float64) string {
	cleanupJournalLock.Lock()
	defer cleanupJournalLock.Unlock()
	if lastRun == 0 {
		return ""
	}
	if lastRun != cleanupJournalRun {
		cleanupJournalRun = lastRun
		cleanupJournalText = RunText(systemctlTimeout, "journalctl", "--unit", nhCleanService,
			"--since", fmt.Sprintf("@%d", int64(lastRun)-5), "--output", "cat", "--no-pager", "--quiet")
	}
	return cleanupJournalText
}

func CollectStorage() Storage {
	show := RunText(systemctlTimeout, "systemctl", "show", nhCleanService, nhCleanTimer,
		"--timestamp=unix", "--property",
		"LoadState,ActiveState,Result,ExecMainStartTimestamp,LastTriggerUSec,NextElapseUSecRealtime")
	units := ParseUnitShow(show)
	journal := ""
	if len(units) >= 2 && units[0]["ActiveState"] != "activating" {
		journal = cleanupJournalFor(CleanupLastRun(units))
	}
	return StorageStateFromParts(StatFS(storagePath), CleanupStateFromParts(show, journal, 0))
}
