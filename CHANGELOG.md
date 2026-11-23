# Changelog

## [0.5.0] - 2026-11-23

### Added

- The app updates itself. It checks for a new release, shows the release notes, downloads the signed update with a progress bar, and offers to relaunch into it. A bad download leaves the running app usable.
- Settings → General holds **Automatic checks** and **Automatic installs**. Automatic checks start off, so a fresh install checks only when you ask.
- The tray menu gains **Check for Updates…**, and the About tab names the newer version when there is one.
- Releases carry signed update archives, delta updates and a manifest. An installed copy and a portable copy both update in place and keep their caches, their configuration and their archived user edits.
- One CI command builds, signs and uploads the installers and the update files, and the changelog writes the update notes and the release body once.

### Removed

- The hand-made app update check that compared version numbers and opened the download page. The updater replaces it.
