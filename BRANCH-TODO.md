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

- [ ] Halloween — October 31 — small pumpkin overlay.
- [ ] Christmas — December 24–26 — small Santa hat overlay.
- [ ] New Year — December 31–January 1 — small sparkle/firework overlay.
- [ ] Sweden National Day — June 6 — small Swedish blue/yellow accent.
- [ ] Ukraine Independence Day — August 24 — small Ukrainian blue/yellow accent.

## Debug controls

Add a small seasonal-effect testing control to the existing Debug tab.

- [ ] Add an **Override seasonal effect** toggle.
- [ ] Add an effect dropdown next to the toggle.
- [ ] Dropdown options: Halloween, Christmas, New Year, Sweden National Day, Ukraine Independence Day.
- [ ] When the toggle is off, normal date-based behavior is used.
- [ ] When the toggle is on, the selected dropdown effect is forced regardless of the current date.
- [ ] Changing either control should update the preview immediately.
- [ ] Keep the debug override frontend-only and session-scoped.
- [ ] Do not add an `Automatic` dropdown option; the toggle itself controls whether the override is active.

## Implementation

- [ ] Add a small centralized seasonal-effect definition/date resolver.
- [ ] Add reusable overlay rendering around the primary Tally logo.
- [ ] Ensure the overlay uses `pointer-events: none` and is hidden from accessibility APIs where appropriate.
- [ ] Add lightweight local assets for each effect; do not add a dependency solely for this feature.
- [ ] Ensure behavior is correct in both light and dark themes.
- [ ] Ensure common sidebar/logo sizes render correctly without clipping or layout shift.

## Tests and validation

- [ ] Add tests for each holiday activation date.
- [ ] Add boundary tests for multi-day Christmas dates.
- [ ] Add tests for the New Year year-boundary behavior.
- [ ] Add a normal-day/no-effect test.
- [ ] Add tests for the debug override toggle and dropdown.
- [ ] Verify an override works regardless of the real current date.
- [ ] Verify turning the override off immediately returns to date-based behavior.
- [ ] Run the project's normal validation/test suite before considering the branch complete.
