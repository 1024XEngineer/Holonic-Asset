import { Bell, CreditCard, Palette, ShieldCheck } from "lucide-react";

import { usePreferencesStore } from "./state/preferences-store";

export function SettingsScreen() {
  const compact = usePreferencesStore((state) => state.compact);
  const notifications = usePreferencesStore((state) => state.notifications);
  const toggleCompact = usePreferencesStore((state) => state.toggleCompact);
  const toggleNotifications = usePreferencesStore(
    (state) => state.toggleNotifications,
  );
  return (
    <main className="settings-page">
      <header>
        <p className="eyebrow">Demo account</p>
        <h1>Settings</h1>
        <p>
          Preferences here are intentionally local until Authentication and
          Workspace are connected.
        </p>
      </header>
      <section className="settings-grid">
        <article>
          <Palette size={20} />
          <div>
            <h2>Workspace density</h2>
            <p>Use a tighter asset grid for larger libraries.</p>
          </div>
          <button
            className={compact ? "toggle toggle--on" : "toggle"}
            onClick={toggleCompact}
            aria-pressed={compact}
          >
            <span />
          </button>
        </article>
        <article>
          <Bell size={20} />
          <div>
            <h2>Generation updates</h2>
            <p>
              Receive a notification when a GenerationRun needs confirmation.
            </p>
          </div>
          <button
            className={notifications ? "toggle toggle--on" : "toggle"}
            onClick={toggleNotifications}
            aria-pressed={notifications}
          >
            <span />
          </button>
        </article>
        <article>
          <CreditCard size={20} />
          <div>
            <h2>Credits</h2>
            <p>Starter plan with 1,280 image credits remaining.</p>
          </div>
          <button className="button button--quiet">Manage</button>
        </article>
        <article>
          <ShieldCheck size={20} />
          <div>
            <h2>Project permissions</h2>
            <p>
              Workspace access will be configured by the Core API authentication
              Module.
            </p>
          </div>
          <span className="version-chip">Demo</span>
        </article>
      </section>
    </main>
  );
}
