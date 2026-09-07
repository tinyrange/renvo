# Unit RollerCAN

The driver implements the complete application I2C register protocol.
See the [NanoC6 example and driver guide](../../../examples/device/rollercan/README.md)
for wiring, API units, firmware compatibility, and build/flash instructions.

Protocol tests live in `frontend_tests/rollercan_driver_test.go` so they are not
embedded in the standalone compiler. Run `go test ./frontend_tests -run '^TestRollerCAN'`
from the repository root.
