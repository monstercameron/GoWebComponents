/* @refresh reload */
import { render } from "solid-js/web";
import "../../../shared/styles.css"; // canonical shared design
import { App } from "./App";

render(() => <App />, document.getElementById("app")!);
