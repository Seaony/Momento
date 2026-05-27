# Momento Cloud Library Sync Phase 0 Status

Status: not complete.

This file tracks the current evidence for the Phase 0 gates in
`2026-05-27-cloud-library-sync-design.md`. It is intentionally separate from the
design so implementation cannot treat the Phase 1 storage-mode foundation as a
release-ready CloudKit integration.

## Current Repository Evidence

| Gate | Current evidence | Status |
| --- | --- | --- |
| macOS deployment target | `Momento.xcodeproj` targets macOS 26. | Known |
| iOS target and deployment target | iOS is in separate repo `/Users/seaony/code/Momento-iOS`; `xcodebuild -list` shows target `Momento`, and `project.pbxproj` has `IPHONEOS_DEPLOYMENT_TARGET = 26.5`. | Partial |
| Final CloudKit container identifier | `/Users/seaony/code/Momento-iOS/Momento/Core/CloudLibrarySyncTypes.swift` uses `iCloud.com.seaony.Momento`; still not verified in Xcode Signing & Capabilities or Apple Developer portal. | Partial |
| macOS CloudKit entitlements | `Momento/Momento.entitlements` has sandbox, file access, network, and Sparkle helper keys only. | Missing |
| iOS CloudKit entitlements | No `.entitlements` file or `CODE_SIGN_ENTITLEMENTS` setting was found in `/Users/seaony/code/Momento-iOS`; CloudKit code exists but signing capability is not verified. | Missing |
| Remote-notification entitlement | `aps-environment` is not present. | Missing |
| Provisioning profiles | Not verified. | Missing |
| CKSyncEngine availability for iOS target | Cannot be confirmed until the iOS deployment target exists. | Missing |
| Real-device CloudKit smoke test | No record save/fetch round trip has been run between Mac and iOS. | Missing |
| Account-switch behavior | Account-state service and tests exist, but no real sign-in/sign-out/account-switch device test has been run. | Partial |
| CKAsset file behavior | Not measured with realistic files. | Missing |
| Large-library scale behavior | Not measured. | Missing |
| Multi-library zone behavior | Not measured. | Missing |
| First-release cloud limits | Seed values exist in the design only; no measured limits. | Missing |
| CloudKit schema/index checklist | Draft exists at `docs/cloudkit-schema-checklist.md`; CloudKit Console indexes are not verified. | Partial |

## Current Implementation Boundary

- `CloudAccountStateService` may check CloudKit account state only when a caller
  explicitly provides a `CKContainer`.
- Cloud library creation remains disabled in the create-library dialog.
- `LibraryStore` still rejects opening, renaming, revealing, or deleting cloud
  placeholders as real libraries.
- macOS now has CloudKit record naming/schema constants for compatibility with
  the iOS repo. No CloudKit metadata records, zones, assets, subscriptions, or
  CKSyncEngine state are implemented on macOS yet.
- Local `.momento` package storage remains the only writable library backend.

## Required Before Enabling Cloud Libraries

1. Keep `/Users/seaony/code/Momento-iOS` as the iOS implementation source and
   do not edit its dirty worktree from the macOS repo task unless explicitly
   requested.
2. Confirm the final CloudKit container identifier in Xcode and Apple Developer
   portal.
3. Wire CloudKit and remote-notification entitlements on both macOS and iOS
   targets.
4. Regenerate and verify provisioning profiles for those entitlements.
5. Confirm CKSyncEngine availability for the chosen iOS target.
6. Run a real-device smoke test proving a record saved on one platform can be
   fetched on the other through the selected private CloudKit container.
7. Run real account-state tests for sign-in, sign-out, account switch,
   `CKContainer.accountStatus`, `CKAccountChanged`, and
   `NSUbiquityIdentityDidChange`.
8. Measure CKAsset upload/download behavior, including file-size failures and
   post-download hash verification.
9. Measure initial sync and local cache behavior with large synthetic libraries.
10. Measure multi-library zone behavior.
11. Write measured v1 limits for file type, file size, library count,
    per-library asset count, and total cloud bytes.
12. Write the CloudKit schema/index checklist before production promotion.

Do not start Phase 2 metadata/blob sync or enable cloud library creation until
the missing Phase 0 evidence above is present.
