// Entry: hydrate the prerendered shell when present, otherwise mount.
import React from "react";
import { createRoot, hydrateRoot } from "react-dom/client";
import App from "./App";
import "./styles/app.css";

const rootEl = document.getElementById("root");
if (!rootEl) throw new Error("missing #root element");

if (rootEl.hasChildNodes()) {
  hydrateRoot(rootEl, React.createElement(React.StrictMode, null, React.createElement(App)));
} else {
  createRoot(rootEl).render(
    React.createElement(React.StrictMode, null, React.createElement(App))
  );
}
