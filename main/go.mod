module main

go 1.22.7

require github.com/cjhouser/fly-io-dist-sys/echo v0.0.0
require github.com/cjhouser/fly-io-dist-sys/uidg v0.0.0
require github.com/cjhouser/fly-io-dist-sys/broadcast v0.0.0
require github.com/cjhouser/fly-io-dist-sys/goc v0.0.0

require github.com/jepsen-io/maelstrom/demo/go v0.0.0-20240813160128-8b9e94c75e59 // indirect

replace github.com/cjhouser/fly-io-dist-sys/echo v0.0.0 => ../echo
replace github.com/cjhouser/fly-io-dist-sys/uidg v0.0.0 => ../uidg
replace github.com/cjhouser/fly-io-dist-sys/broadcast v0.0.0 => ../broadcast
replace github.com/cjhouser/fly-io-dist-sys/goc v0.0.0 => ../goc
