import { render } from "preact";
import "../../../shared/styles.css"; // canonical shared design
import { App } from "./app";

render(<App />, document.getElementById("app")!);
