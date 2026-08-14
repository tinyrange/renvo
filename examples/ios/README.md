# iOS CompilerJIT target

[`ios_arm64.rtg`](../../backends/ios_arm64.rtg) is the compiler-side iOS port.
It reuses the shared AArch64 and
Darwin lowering through an external CompilerJIT definition, then specializes
the generated Mach-O contract for iOS. The Renvo compile does not require Xcode
or an iPhoneOS SDK.

The current image contract provides:

- a PIE AArch64 Mach-O executable
- `LC_BUILD_VERSION` for iOS 13.0
- 16 KiB Mach-O segments without writable/executable overlap
- the ordinary `/usr/lib/libSystem.B.dylib` runtime binding
- an ad-hoc SHA-256 code directory refreshed after the iOS header is written
- `ios`, `darwin`, `unix`, `aarch64`, and `arm64` frontend build tags
- UIKit application/delegate startup without an Xcode-generated launcher
- a CoreGraphics-backed software surface presented through `UIImageView`
- single-touch pointer input bridged into ordinary Forms events

The compiler output is an executable, not a complete application bundle.
Installing it on a physical device additionally requires an `Info.plist`, a
development provisioning profile, matching entitlements, and an Apple-issued
signature.

## Host-independent checks

Run the Mach-O and Forms target tests with:

```sh
go test ./internal/backendjit \
  -run 'TestCompilerJITIOSARM64(MachO|Forms|Controls)Image' \
  -count=1 -v
```

For a small compiler smoke test:

```sh
go run ./cmd/renvo \
  -backend backends/ios_arm64.rtg \
  -t ios/arm64 \
  -s -o sandbox/renvo-ios \
  internal/backendjit/testdata/mobile_entry.go
```

## Example applications

The hello application is a small 360 x 800 retained Forms layout:

```sh
go run ./cmd/renvo \
  -backend backends/ios_arm64.rtg \
  -t ios/arm64 \
  -s -o sandbox/RenvoForms \
  ./examples/forms_hello
```

The full controls gallery is shared directly with Android. Platform build tags
only specialize its subtitle, so both targets compile the same controls and
event handlers:

```sh
go run ./cmd/renvo \
  -backend backends/ios_arm64.rtg \
  -t ios/arm64 \
  -s -o sandbox/RenvoControls \
  ./examples/forms_controls
```

The remaining sections build, sign, install, and launch that controls gallery.

## Physical-device prerequisites

The verified command-line workflow uses:

- macOS with the standard `security`, `codesign`, `plutil`, and `zip` tools
- an Apple Developer account that can create development certificates and
  provisioning profiles
- an ARM64 iPhone registered in that developer team
- Developer Mode enabled on the iPhone
- a trusted USB pairing between the Mac and iPhone
- Python 3 and `pymobiledevice3` for installation and developer services

Create an isolated Python environment if `pymobiledevice3` is not already
available:

```sh
python3 -m venv sandbox/pymobiledevice3-venv
sandbox/pymobiledevice3-venv/bin/python -m pip install -U pip pymobiledevice3
```

Connect and unlock the iPhone, accept its Trust prompt, then confirm that it is
visible:

```sh
sandbox/pymobiledevice3-venv/bin/pymobiledevice3 usbmux list
```

The returned `Identifier` is the device UDID used by later commands.

## Apple certificate and provisioning profile

The certificate, private key, provisioning profile, and device registration
must all belong to the same Apple Developer team.

1. In Certificates, Identifiers & Profiles on the Apple Developer website,
   create an **Apple Development** certificate from a certificate-signing
   request. Keychain Access can generate the request and retain its private key.
2. Register the connected iPhone using the UDID reported by `usbmux list`.
3. Create an explicit App ID for the intended bundle identifier, such as
   `dev.renvo.forms`. A wildcard App ID also works when its pattern contains the
   concrete identifier.
4. Create an **iOS App Development** provisioning profile. Select the App ID,
   Apple Development certificate, and connected iPhone.
5. Download the `.mobileprovision` file. Install the development certificate
   and its private key in the login keychain.

Keep private keys and device-specific profiles under ignored `sandbox/` storage
or another protected directory. Never commit them.

Confirm that macOS can see the signing identity:

```sh
security find-identity -v -p codesigning
```

Decode the profile and inspect its team, application pattern, certificate, and
registered devices:

```sh
RENVO_IOS_PROFILE=/absolute/path/Renvo_iOS_Development.mobileprovision
RENVO_IOS_ROOT="$PWD/sandbox/ios-controls"

mkdir -p "$RENVO_IOS_ROOT"
security cms -D -i "$RENVO_IOS_PROFILE" \
  -o "$RENVO_IOS_ROOT/Profile.plist"
plutil -p "$RENVO_IOS_ROOT/Profile.plist"
```

In particular, verify these profile fields:

- `ExpirationDate` is in the future.
- `ProvisionedDevices` contains the connected iPhone UDID.
- `Entitlements:get-task-allow` is true.
- `Entitlements:application-identifier` is either the concrete
  `TEAM_ID.bundle.identifier` or a matching wildcard.
- `DeveloperCertificates` contains the Apple Development certificate selected
  by `codesign`.

## Build the application bundle

Set deployment-specific values. The signing identity can be either the full
certificate name or its SHA-1 hash from `security find-identity`:

```sh
RENVO_IOS_BUNDLE_ID=dev.renvo.forms
RENVO_IOS_IDENTITY='Apple Development: Your Name (XXXXXXXXXX)'
RENVO_IOS_DEVICE=00000000-0000000000000000
RENVO_MOBILE="$PWD/sandbox/pymobiledevice3-venv/bin/pymobiledevice3"
RENVO_IOS_APP="$RENVO_IOS_ROOT/Payload/RenvoForms.app"
RENVO_IOS_IPA="$RENVO_IOS_ROOT/RenvoControls.ipa"
```

Create the bundle and compile the shared controls gallery directly into its
executable slot:

```sh
mkdir -p "$RENVO_IOS_APP"

go run ./cmd/renvo \
  -backend backends/ios_arm64.rtg \
  -t ios/arm64 \
  -s -o "$RENVO_IOS_APP/RenvoForms" \
  ./examples/forms_controls

install -m 644 examples/ios/Info.plist "$RENVO_IOS_APP/Info.plist"
install -m 644 "$RENVO_IOS_PROFILE" \
  "$RENVO_IOS_APP/embedded.mobileprovision"
chmod 755 "$RENVO_IOS_APP/RenvoForms"
plutil -replace CFBundleIdentifier -string "$RENVO_IOS_BUNDLE_ID" \
  "$RENVO_IOS_APP/Info.plist"
```

`CFBundleExecutable` must remain `RenvoForms`, matching the filename above. The
provided plist keeps the system status bar visible and declares portrait phone
support.

Inspect the unsigned compiler artifact when changing the target:

```sh
file "$RENVO_IOS_APP/RenvoForms"
otool -hv "$RENVO_IOS_APP/RenvoForms"
otool -L "$RENVO_IOS_APP/RenvoForms"
vtool -show-build "$RENVO_IOS_APP/RenvoForms"
```

The expected result is an ARM64 executable, iOS platform 2, minimum iOS 13.0,
and load commands for libSystem, libobjc, UIKit, and CoreGraphics.

## Derive concrete development entitlements

Read the team identifier from the decoded profile:

```sh
RENVO_IOS_TEAM_ID=$(/usr/libexec/PlistBuddy \
  -c 'Print :Entitlements:com.apple.developer.team-identifier' \
  "$RENVO_IOS_ROOT/Profile.plist")
RENVO_IOS_ENTITLEMENTS="$RENVO_IOS_ROOT/Entitlements.plist"
```

Create a minimal entitlement set whose application identifier is concrete, not
the profile's wildcard:

```sh
plutil -create xml1 "$RENVO_IOS_ENTITLEMENTS"
plutil -insert application-identifier \
  -string "$RENVO_IOS_TEAM_ID.$RENVO_IOS_BUNDLE_ID" \
  "$RENVO_IOS_ENTITLEMENTS"
plutil -insert 'com\.apple\.developer\.team-identifier' \
  -string "$RENVO_IOS_TEAM_ID" \
  "$RENVO_IOS_ENTITLEMENTS"
plutil -insert get-task-allow -bool YES "$RENVO_IOS_ENTITLEMENTS"
plutil -p "$RENVO_IOS_ENTITLEMENTS"
```

The bundle identifier, `application-identifier`, certificate team, and profile
team must agree. A mismatch normally installs as a profile or entitlement
validation failure.

## Sign and validate

The Renvo compiler emits an ad-hoc signature so the standalone Mach-O has a
structurally valid code directory. Physical iOS deployment must replace it with
the Apple Development signature:

```sh
codesign --force --verbose=4 \
  --sign "$RENVO_IOS_IDENTITY" \
  --entitlements "$RENVO_IOS_ENTITLEMENTS" \
  --timestamp=none \
  --generate-entitlement-der \
  "$RENVO_IOS_APP"
```

Validate both the resource seal and final entitlements before installation:

```sh
codesign --verify --deep --strict --verbose=4 "$RENVO_IOS_APP"
codesign -dvvv "$RENVO_IOS_APP"
codesign -d --entitlements :- "$RENVO_IOS_APP"
```

The detailed output should name the Apple Development authority, set
`TeamIdentifier`, and report the concrete bundle identifier. `Signature=adhoc`
or `TeamIdentifier=not set` is not deployable to a stock iPhone.

### Non-interactive signing keychain

If `codesign` waits for private-key access or returns `errSecInternalComponent`,
approve `/usr/bin/codesign` in the private key's Keychain Access controls. For
automation, use an isolated keychain and temporarily add it to the search list:

```sh
RENVO_SIGN_KEYCHAIN="$RENVO_IOS_ROOT/renvo-signing.keychain-db"
RENVO_SIGN_PASSWORD='replace-with-a-local-keychain-password'
RENVO_SIGN_P12=/absolute/path/AppleDevelopment.p12
RENVO_SIGN_P12_PASSWORD='replace-with-the-p12-password'
RENVO_LOGIN_KEYCHAIN=$(security default-keychain -d user | tr -d ' "')

security create-keychain -p "$RENVO_SIGN_PASSWORD" "$RENVO_SIGN_KEYCHAIN"
security set-keychain-settings -lut 21600 "$RENVO_SIGN_KEYCHAIN"
security unlock-keychain -p "$RENVO_SIGN_PASSWORD" "$RENVO_SIGN_KEYCHAIN"
security import "$RENVO_SIGN_P12" \
  -k "$RENVO_SIGN_KEYCHAIN" \
  -P "$RENVO_SIGN_P12_PASSWORD" \
  -T /usr/bin/codesign
security set-key-partition-list \
  -S 'apple-tool:,apple:,codesign:' \
  -s -k "$RENVO_SIGN_PASSWORD" "$RENVO_SIGN_KEYCHAIN"
security default-keychain -d user -s "$RENVO_SIGN_KEYCHAIN"
security list-keychains -d user \
  -s "$RENVO_SIGN_KEYCHAIN" "$RENVO_LOGIN_KEYCHAIN"
```

Run `codesign`, then restore and lock the isolated keychain:

```sh
security default-keychain -d user -s "$RENVO_LOGIN_KEYCHAIN"
security list-keychains -d user -s "$RENVO_LOGIN_KEYCHAIN"
security lock-keychain "$RENVO_SIGN_KEYCHAIN"
```

Do not leave a private signing key in an unlocked automation keychain.

## Package and install

An IPA is a zip archive whose root contains exactly one `Payload/*.app` bundle:

```sh
(cd "$RENVO_IOS_ROOT" && zip -qry RenvoControls.ipa Payload)
unzip -l "$RENVO_IOS_IPA"
```

Install it over USB:

```sh
"$RENVO_MOBILE" apps install \
  --udid "$RENVO_IOS_DEVICE" \
  --developer \
  "$RENVO_IOS_IPA"
```

`pymobiledevice3` can also install the `.app` directory directly. Listing the
installed application is a useful profile check:

```sh
"$RENVO_MOBILE" apps list \
  --udid "$RENVO_IOS_DEVICE" \
  --type User
```

Find `dev.renvo.forms` in the result and verify `ProfileValidated: true`, the
expected `SignerIdentity`, and the concrete entitlements.

## Mount developer services and launch

Developer Mode must be enabled before developer services can launch or inspect
the app:

```sh
"$RENVO_MOBILE" mounter query-developer-mode-status \
  --udid "$RENVO_IOS_DEVICE"
```

On iOS 17 and newer, mount the personalized Developer Disk Image through a
userspace tunnel. `auto-mount` obtains the correct image for the connected
device:

```sh
"$RENVO_MOBILE" mounter auto-mount \
  --udid "$RENVO_IOS_DEVICE" \
  --userspace
```

Launch using CoreDevice. The final empty argument satisfies the current CLI's
variadic application-argument parameter:

```sh
"$RENVO_MOBILE" developer core-device launch-application \
  --userspace \
  "$RENVO_IOS_BUNDLE_ID" ''
```

The DVT launch path is also available and allows an explicit UDID:

```sh
"$RENVO_MOBILE" developer dvt launch \
  --udid "$RENVO_IOS_DEVICE" \
  --userspace \
  "$RENVO_IOS_BUNDLE_ID"
```

Confirm that the process remains alive:

```sh
"$RENVO_MOBILE" developer dvt process-id-for-bundle-id \
  --udid "$RENVO_IOS_DEVICE" \
  --userspace \
  "$RENVO_IOS_BUNDLE_ID"
```

A nonzero PID means the process is running. Capture the rendered screen with:

```sh
"$RENVO_MOBILE" developer dvt screenshot \
  --udid "$RENVO_IOS_DEVICE" \
  --userspace \
  "$RENVO_IOS_ROOT/renvo-controls.png"
```

## Debugging

### Installation failures

For `ApplicationVerificationFailed`, `PackageInspectionFailed`, or provisioning
errors:

1. Run the three `codesign` validation commands again.
2. Decode the embedded profile, not a similarly named profile elsewhere.
3. Confirm that the profile contains the device UDID and selected certificate.
4. Confirm that the bundle ID and concrete application entitlement match.
5. Ensure the IPA contains only one `.app` beneath `Payload/`; backup bundles
   inside `Payload/` cause package inspection to reject the archive.

### Launch failures

CoreDevice usually reports a deeper SpringBoard or RunningBoard error than the
DVT launcher. `NSPOSIXErrorDomain` code 85 is `EBADEXEC`, meaning iOS rejected
the executable or its signature before `main` ran.

Check the installed process and local artifact first:

```sh
"$RENVO_MOBILE" developer dvt process-id-for-bundle-id \
  --udid "$RENVO_IOS_DEVICE" --userspace "$RENVO_IOS_BUNDLE_ID"
codesign --verify --deep --strict --verbose=4 "$RENVO_IOS_APP"
vtool -show-build "$RENVO_IOS_APP/RenvoForms"
```

An Apple Development signature must replace the compiler's ad-hoc signature.
Rebuild a clean `.app` directory before signing if a failed signing attempt left
`*.cstemp` files or stale `_CodeSignature` resources.

### Device logs

Start the USB syslog stream before reproducing a launch or input problem:

```sh
"$RENVO_MOBILE" syslog live \
  --udid "$RENVO_IOS_DEVICE" \
  --label \
  -ei 'RenvoForms|dev\.renvo|amfid|dyld|launch|signature'
```

Useful sources include SpringBoard, RunningBoard, `amfid`, `trustd`, `dyld`, and
the kernel. Signature rejection before process creation will not produce Renvo
application logs, so retain the surrounding system messages.

### Rendering and input

If the process remains alive but the UI is blank, capture a screenshot and
check for UIKit/CoreGraphics errors in syslog. The Forms app uses a 360 x 800
logical surface scaled to the device view. Touch coordinates are mapped back to
that logical surface. The current bridge is single-touch; multi-touch gestures
and native dialogs are not implemented yet.

## Verified device workflow

This flow has been verified on a physical ARM64 iPhone through native Apple
Development signing, profile validation, USB installation, personalized
Developer Disk Image mounting, CoreDevice launch, full controls rendering, text
entry, tabs, lists, and touch interaction.
