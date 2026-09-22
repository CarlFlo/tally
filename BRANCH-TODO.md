# Seasonal Logo Overlays

Branch: `feature/seasonal-logo-overlays`

## Goal

Add small seasonal/holiday decorations as non-interactive overlays on the existing Tally logo. Keep the feature frontend-only, lightweight, and intentionally limited in scope.

## Agreed behavior

- Keep the existing Tally logo unchanged.
- Render seasonal artwork as a small overlay positioned on the logo.
- The overlay must not change layout, logo dimensions, navigation behavior, or click targets.
- The overlay should be non-interactive and decorative only.
- Use the browser/local calendar date for normal date-based activation.
- Do not add a holiday library, backend scheduler, database state, locale inference, or general holiday configuration system.
- Keep the first version static rather than animated.

## Initial seasonal effects

- [x] Halloween — October 31 — small pumpkin overlay.
- [x] Christmas — December 24–26 — small Santa hat overlay.
- [x] New Year — December 31–January 1 — small sparkle/firework overlay.
- [x] Sweden National Day — June 6 — small Swedish blue/yellow accent.
- [x] Ukraine Independence Day — August 24 — small Ukrainian blue/yellow accent.

## Debug controls

Add a small seasonal-effect testing control to the existing Debug tab.

- [x] Add an **Override seasonal effect** toggle.
- [x] Add an effect dropdown next to the toggle.
- [x] Dropdown options: Halloween, Christmas, New Year, Sweden National Day, Ukraine Independence Day.
- [x] When the toggle is off, normal date-based behavior is used.
- [x] When the toggle is on, the selected dropdown effect is forced regardless of the current date.
- [x] Changing either control should update the preview immediately.
- [x] Keep the debug override frontend-only and session-scoped.
- [x] Do not add an `Automatic` dropdown option; the toggle itself controls whether the override is active.

## Implementation

- [x] Add a small centralized seasonal-effect definition/date resolver.
- [x] Add reusable overlay rendering around the primary Tally logo.
- [x] Ensure the overlay uses `pointer-events: none` and is hidden from accessibility APIs where appropriate.
- [x] Add lightweight local assets for each effect; do not add a dependency solely for this feature.
- [x] Ensure behavior is correct in both light and dark themes.
- [x] Ensure common sidebar/logo sizes render correctly without clipping or layout shift.

## Tests and validation

- [x] Add tests for each holiday activation date.
- [x] Add boundary tests for multi-day Christmas dates.
- [x] Add tests for the New Year year-boundary behavior.
- [x] Add a normal-day/no-effect test.
- [x] Add tests for the debug override toggle and dropdown.
- [x] Verify an override works regardless of the real current date.
- [x] Verify turning the override off immediately returns to date-based behavior.
Final merge gate: the authoritative `Checks` push run for the exact final branch HEAD must complete successfully.
