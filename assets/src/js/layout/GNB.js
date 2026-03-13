import * as $ from "bm.js/bm.module.js";
//import {html, render} from '/lib/lit-html/lit-html.js';
import {html, render} from 'lit-html';
import {css} from "@/common.js";

class GlobalNavigationBar extends HTMLElement {
	template() {
		return html`
			<style>
				${css}

				:host {
					display: flex;
					justify-content: space-between;
					align-items: flex-end;
				}
				a {
					color: inherit;
					text-decoration: none;
				}
				a:hover {
					text-decoration: underline;
				}

				#logo {
					font-size: 1.4rem;
				}
			</style>
			<section id="logo">
				<a href="/">grim</a>
			</section>
			<section id="action">
				<a href="/editor">Editor</a>
				<a href="/guide">Guide</a>
				<a href="https://github.com/bluemir/grim"><c-icon fa kind="link" size="1rem" /></a>
			</section>
		`;
	}
	constructor() {
		super();

		this.attachShadow({mode: 'open'})
	}

	async render() {
		render(this.template(), this.shadowRoot);
	}

	async onConnected() {

		this.render();
	}
}
customElements.define("global-navigation-bar", GlobalNavigationBar);

