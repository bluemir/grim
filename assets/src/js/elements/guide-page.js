import * as $ from "bm.js/bm.module.js";
import { html, render } from "lit-html";
import { css } from "@/common.js";

class GuidePage extends HTMLElement {
	#manifest = null;
	#slug = null;
	#pageHTML = "";

	get #currentSlug() {
		const path = location.pathname;
		const prefix = "/guide/";
		if (!path.startsWith(prefix)) return null;
		return path.substring(prefix.length) || null;
	}

	template() {
		const manifest = this.#manifest;
		const slug = this.#slug;

		return html`
			<style>
				${css}

				:host {
					display: flex;
					flex: 1;
					min-height: 0;
					overflow: hidden;
				}

				#guide-nav {
					width: 220px;
					flex-shrink: 0;
					overflow-y: auto;
					border-right: 1px solid var(--gray-300);
					padding: 1rem 0;
					background: var(--gray-100);
				}

				#guide-nav h3 {
					font-size: 0.75rem;
					font-weight: 600;
					text-transform: uppercase;
					letter-spacing: 0.05em;
					color: var(--gray-500);
					padding: 0.75rem 1rem 0.25rem;
					margin: 0;
				}

				#guide-nav a {
					display: block;
					padding: 0.3rem 1rem 0.3rem 1.5rem;
					font-size: 0.875rem;
					color: inherit;
					text-decoration: none;
					border-left: 3px solid transparent;
				}

				#guide-nav a:hover {
					background: var(--gray-200);
				}

				#guide-nav a.active {
					border-left-color: var(--blue-500);
					font-weight: 600;
					color: var(--blue-700);
				}

				#guide-content {
					flex: 1;
					overflow-y: auto;
					padding: 2rem;
					/*max-width: 800px;*/
				}

				#guide-content h1 {
					font-size: 1.75rem;
					margin-bottom: 1rem;
					border-bottom: 1px solid var(--gray-200);
					padding-bottom: 0.5rem;
				}
				#guide-content h2 { font-size: 1.3rem; margin: 2rem 0 0.75rem; }
				#guide-content h3 { font-size: 1rem; margin: 1.5rem 0 0.5rem; }
				#guide-content p { line-height: 1.7; margin-bottom: 1rem; }
				#guide-content ul, #guide-content ol { padding-left: 1.5rem; margin-bottom: 1rem; line-height: 1.7; }
				#guide-content table { border-collapse: collapse; width: 100%; margin-bottom: 1rem; font-size: 0.9rem; }
				#guide-content th, #guide-content td { border: 1px solid var(--gray-300); padding: 0.4rem 0.75rem; text-align: left; }
				#guide-content th { background: var(--gray-100); font-weight: 600; }
				#guide-content code { background: var(--gray-100); border-radius: 3px; padding: 0.1em 0.3em; font-family: monospace; font-size: 0.9em; }
				#guide-content pre { background: var(--gray-900); color: var(--gray-100); border-radius: 6px; padding: 1rem; overflow-x: auto; margin-bottom: 1rem; }
				#guide-content pre code { background: none; padding: 0; font-size: 0.875rem; }
				#guide-content blockquote { border-left: 4px solid var(--blue-300); padding: 0.5rem 1rem; margin: 1rem 0; background: var(--blue-50); color: var(--gray-700); border-radius: 0 4px 4px 0; }
				#guide-content a { color: var(--blue-700); }
				#guide-content hr { border: none; border-top: 1px solid var(--gray-200); margin: 2rem 0; }
			</style>

			<nav id="guide-nav">
				${manifest ? manifest.sections.map(section => html`
					<h3>${section.title}</h3>
					${section.items.map(item => html`
						<a href="/guide/${item.slug}"
						   class=${item.slug === slug ? "active" : ""}
						   @click=${(e) => this.#navigate(e, item.slug)}>
							${item.title}
						</a>
					`)}
				`) : html`<p style="padding:1rem;color:var(--gray-500)">로딩 중...</p>`}
			</nav>

			<article id="guide-content" .innerHTML=${this.#pageHTML}></article>
		`;
	}

	constructor() {
		super();
		this.attachShadow({ mode: "open" });
	}

	async render() {
		render(this.template(), this.shadowRoot);
	}

	async onConnected() {
		window.addEventListener("popstate", () => this.#loadPage(this.#currentSlug));
		await this.#loadAll(this.#currentSlug);
	}

	async #loadAll(slug) {
		const [manifestRes, pageRes] = await Promise.all([
			$.request("GET", "/api/v1/guide/manifest"),
			$.request("GET", `/api/v1/guide/pages/${slug}`),
		]);
		this.#manifest = manifestRes.json;
		this.#slug = slug;
		this.#pageHTML = pageRes.json.html ?? "";
		await this.render();
	}

	async #loadPage(slug) {
		const pageRes = await $.request("GET", `/api/v1/guide/pages/${slug}`);
		this.#slug = slug;
		this.#pageHTML = pageRes.json.html ?? "";
		await this.render();
	}

	#navigate(e, slug) {
		e.preventDefault();
		history.pushState(null, "", `/guide/${slug}`);
		this.#loadPage(slug);
	}
}
customElements.define("guide-page", GuidePage);
