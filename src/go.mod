module arrangement

go 1.25.0

require golang.org/x/image v0.41.0 // indirect

require github.com/federico-pepe/ableton-push-hack/core v0.0.0

require golang.org/x/text v0.37.0 // indirect

// Dev only: local checkout of the framework core. Pin a real version before release.
replace github.com/federico-pepe/ableton-push-hack/core => ../../../Documents/GitHub/ableton-push-hack/core
