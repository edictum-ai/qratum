import { Component, Suspense, lazy, type ReactNode } from "react";

// The chosen Qratum landing direction: "Light Ledger" (dark-ledger's design +
// full animation set, recolored to a light/paper surface).
const LedgerLightPage = lazy(() => import("./drafts/ledger-light/Page"));

class Boundary extends Component<{ children: ReactNode }, { error: Error | null }> {
  state = { error: null as Error | null };
  static getDerivedStateFromError(error: Error) {
    return { error };
  }
  render() {
    if (this.state.error) {
      return (
        <div style={{ padding: "4rem", fontFamily: "JetBrains Mono, monospace" }}>
          <p style={{ color: "#b00" }}>This page errored:</p>
          <pre style={{ whiteSpace: "pre-wrap" }}>{String(this.state.error?.message ?? this.state.error)}</pre>
        </div>
      );
    }
    return this.props.children;
  }
}

export default function App() {
  return (
    <Boundary>
      <Suspense
        fallback={
          <div
            style={{
              minHeight: "100vh",
              display: "grid",
              placeItems: "center",
              opacity: 0.5,
              fontFamily: "JetBrains Mono, monospace",
            }}
          >
            loading qratum…
          </div>
        }
      >
        <LedgerLightPage />
      </Suspense>
    </Boundary>
  );
}
