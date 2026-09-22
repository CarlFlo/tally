import { useActiveSeasonalEffect } from "./seasonalEffects";

export function Logo() {
  const seasonalEffect = useActiveSeasonalEffect();

  return (
    <div className="logo">
      <span className="logo-mark-wrap">
        <span className="logo-mark">
          <i />
          <i />
          <i />
          <b />
        </span>
        {seasonalEffect && (
          <span
            className={`seasonal-logo-overlay seasonal-logo-overlay--${seasonalEffect.id}`}
            data-seasonal-effect={seasonalEffect.id}
            aria-hidden="true"
          >
            {seasonalEffect.renderOverlay()}
          </span>
        )}
      </span>
      <span>
        tally<span className="logo-period">.</span>
      </span>
    </div>
  );
}
