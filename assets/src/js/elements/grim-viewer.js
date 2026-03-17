// <grim-viewer> — standalone custom element for embedding grim diagrams.
// No external dependencies (no lit-html, no bm.js).
// Usage:
//   <script type="module" src="https://grim.bluemir.me/static/elements/grim-viewer.js"></script>
//   <grim-viewer>A -> B</grim-viewer>

// Derive base URL from this module's location.
// e.g. .../static/{rev}/js/elements/grim-viewer.js -> .../static/{rev}/
const baseURL = new URL("../../", import.meta.url).href;

// Singleton WASM loading — shared across all <grim-viewer> instances.
const wasmReady = (async () => {
	// Load wasm_exec.js (Go runtime support) via <script> tag.
	if (typeof Go === "undefined") {
		await new Promise((resolve, reject) => {
			const script = document.createElement("script");
			script.src = `${baseURL}bundle/wasm/wasm_exec.js`;
			script.onload = resolve;
			script.onerror = () => reject(new Error("Failed to load wasm_exec.js"));
			document.head.appendChild(script);
		});
	}

	const go = new Go();
	const result = await WebAssembly.instantiateStreaming(
		fetch(`${baseURL}bundle/wasm/grim.wasm`),
		go.importObject,
	);
	go.run(result.instance);
})();

class GrimViewer extends HTMLElement {
	#shadow;
	#observer;
	#debounceTimer;

	constructor() {
		super();
		this.#shadow = this.attachShadow({ mode: "open" });
		this.#shadow.innerHTML = `
			<style>
				:host {
					display: block;
				}
				#container {
					display: inline-block;
				}
			</style>
			<div id="container"></div>
		`;
	}

	async connectedCallback() {
		await wasmReady;
		this.#render();

		// Watch for textContent changes.
		this.#observer = new MutationObserver(() => {
			clearTimeout(this.#debounceTimer);
			this.#debounceTimer = setTimeout(() => this.#render(), 100);
		});
		this.#observer.observe(this, {
			childList: true,
			characterData: true,
			subtree: true,
		});
	}

	disconnectedCallback() {
		if (this.#observer) {
			this.#observer.disconnect();
			this.#observer = null;
		}
		clearTimeout(this.#debounceTimer);
	}

	#render() {
		const source = this.textContent.trim();
		const container = this.#shadow.getElementById("container");
		if (!source) {
			container.innerHTML = "";
			return;
		}
		try {
			const svg = grimRender(source);
			container.innerHTML = svg;
		} catch (e) {
			console.error("<grim-viewer> render error:", e);
		}
	}
}

customElements.define("grim-viewer", GrimViewer);
