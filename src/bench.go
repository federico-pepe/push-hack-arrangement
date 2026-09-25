package main

// -bench <seconds>: take over the display and redraw at the normal rate while a
// fake playhead runs, then print the CPU used by this process. For measuring on
// the device (use with -set <file.als>). Shows a moving view on the Push for the duration.

import (
	"fmt"
	"image"
	"syscall"
	"time"

	"github.com/federico-pepe/ableton-push-hack/core/pmclient"
)

func cpuSeconds() float64 {
	var ru syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	tv := func(t syscall.Timeval) float64 { return float64(t.Sec) + float64(t.Usec)/1e6 }
	return tv(ru.Utime) + tv(ru.Stime)
}

func runBench(pm *pmclient.Client, s *Set, vc *viewCtl, secs int) error {
	if err := pm.SetMode(2); err != nil {
		return err
	}
	defer func() { _ = pm.SetMode(0) }()
	vc.zoomV(30)
	vc.zoomH(20)
	cpu0, t0 := cpuSeconds(), time.Now()
	frames := 0
	for time.Since(t0) < time.Duration(secs)*time.Second {
		play := time.Since(t0).Seconds() * 2 // 2 beats per second
		vc.follow(play, true)
		cp := *s
		cp.Playhead = play
		var img image.Image = renderArrangement(&cp, vc.viewport(&cp))
		if err := pm.PushImage(img); err != nil {
			return err
		}
		frames++
		time.Sleep(minPushGap)
	}
	wall := time.Since(t0).Seconds()
	cpu := cpuSeconds() - cpu0
	fmt.Printf("bench: %d frames in %.1fs (%.1f fps), hack CPU %.2fs = %.1f%% of one core\n",
		frames, wall, float64(frames)/wall, cpu, 100*cpu/wall)
	return nil
}
