import * as $ from "bm.js/bm.module.js";
import {html, render} from 'lit-html';
import {css} from "@/common.js";

class Icon extends HTMLElement {
	template() {
		const kind = this.getAttribute("kind");
		const fa = this.getAttribute("fa");
		const hasFa = this.hasAttribute("fa");

		let iconContent;
		if (hasFa && kind) {
			const style = (fa === "" || fa === null || fa === "solid") ? "fa-solid" : `fa-${fa}`;
			iconContent = html`<i class="${style} fa-${kind}"></i>`;
		} else {
			iconContent = html`<span class="material-symbols-outlined">${kind}</span>`;
		}

		return html`
			<style>
				${css}
				:host { display: inline-flex; }
				span.material-symbols-outlined {
					${this.size}
					cursor: default;
					vertical-align: bottom;
				}
				i { ${this.size} }
			</style>
			${iconContent}
		`;
	}
	constructor() {
		super();

		this.attachShadow({mode:'open'});
	}
	static get observedAttributes() {
		return ["kind", "size", "fa"];
	}
	onAttributeChanged(name, old, v) {
		this.render();
	}
	async render() {
		render(this.template(), this.shadowRoot);
	}
	// attribute
	get size() {
		let n = this.attr("size");
		return n ? `font-size: ${n};` : ""
	}
}
customElements.define("c-icon", Icon);
