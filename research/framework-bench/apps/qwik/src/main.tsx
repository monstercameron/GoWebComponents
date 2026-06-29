import { render } from "@builder.io/qwik";
import "../../../shared/styles.css"; // canonical shared design
import { App } from "./app";

render(document.getElementById("app")!, <App />);
