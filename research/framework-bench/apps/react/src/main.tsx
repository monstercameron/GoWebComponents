import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "../../../shared/styles.css"; // canonical shared design
import { App } from "./App";

createRoot(document.getElementById("app")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
