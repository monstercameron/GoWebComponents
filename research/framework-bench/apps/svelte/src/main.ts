import { mount } from "svelte";
import "../../../shared/styles.css"; // canonical shared design
import App from "./App.svelte";

export default mount(App, { target: document.getElementById("app")! });
