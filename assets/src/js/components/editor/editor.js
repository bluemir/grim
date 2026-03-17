class CEditor extends HTMLElement {
	static get observedAttributes() {
		return ["highlight"];
	}

	constructor() {
		super();
		this.attachShadow({ mode: "open" });
		this.shadowRoot.innerHTML = `
			<style>
				:host {
					display: flex;
					flex-direction: column;
					flex: 1;
					min-height: 0;
					position: relative;
				}
				::slotted(textarea) {
					flex: 1;
					min-height: 0;
				}
				#highlight-overlay {
					position: absolute;
					pointer-events: none;
					overflow: hidden;
					z-index: 1;
				}
			</style>
			<slot></slot>
			<div id="highlight-overlay"></div>
		`;
		this._overlay = this.shadowRoot.getElementById("highlight-overlay");
		this._highlightRanges = null;
		this._textarea = null;
		this._onScroll = () => this._updateHighlightPositions();
	}

	connectedCallback() {
		const slot = this.shadowRoot.querySelector("slot");
		slot.addEventListener("slotchange", () => this._bindTextarea());
		this._bindTextarea();
	}

	disconnectedCallback() {
		if (this._textarea) {
			this._textarea.removeEventListener("scroll", this._onScroll);
			this._textarea = null;
		}
	}

	_bindTextarea() {
		if (this._textarea) {
			this._textarea.removeEventListener("scroll", this._onScroll);
		}
		this._textarea = this.querySelector("textarea");
		if (!this._textarea) return;

		const cs = getComputedStyle(this._textarea);
		if (cs.lineHeight === "normal") {
			this._textarea.style.lineHeight = "1.2em";
		}

		this._textarea.addEventListener("scroll", this._onScroll);
		this._syncOverlayInset();
		this._renderHighlights();
	}

	_syncOverlayInset() {
		if (!this._textarea) return;
		const cs = getComputedStyle(this._textarea);
		this._overlay.style.top = cs.borderTopWidth;
		this._overlay.style.bottom = cs.borderBottomWidth;
		this._overlay.style.left = cs.borderLeftWidth;
		this._overlay.style.right = cs.borderRightWidth;
	}

	attributeChangedCallback(name, oldVal, newVal) {
		if (name === "highlight") {
			this._renderHighlights();
		}
	}

	_renderHighlights() {
		const attr = this.getAttribute("highlight");
		if (!attr) {
			this._overlay.innerHTML = "";
			this._highlightRanges = null;
			return;
		}

		this._highlightRanges = this._parseLineRanges(attr);
		this._overlay.innerHTML = "";
		for (const range of this._highlightRanges) {
			const bar = document.createElement("div");
			bar.style.cssText =
				"position:absolute;left:0;right:0;" +
				"pointer-events:none;background:rgba(74,144,226,0.15);";
			this._overlay.appendChild(bar);
		}
		this._updateHighlightPositions();
	}

	_updateHighlightPositions() {
		if (!this._textarea || !this._highlightRanges) return;
		const ta = this._textarea;
		const bars = this._overlay.children;
		const lines = ta.value.split('\n');
		const paddingBottom = parseFloat(getComputedStyle(ta).paddingBottom);

		const mirror = this._getMirror();
		for (let i = 0; i < this._highlightRanges.length; i++) {
			const { start, end } = this._highlightRanges[i];
			// scrollHeight = paddingTop + contentHeight + paddingBottom
			// Subtract paddingBottom to get position relative to padding-box top
			mirror.textContent = start > 1 ? lines.slice(0, start - 1).join('\n') : '';
			const topPos = mirror.scrollHeight - paddingBottom;

			mirror.textContent = lines.slice(0, end).join('\n');
			const bottomPos = mirror.scrollHeight - paddingBottom;

			bars[i].style.top = (topPos - ta.scrollTop) + "px";
			bars[i].style.height = (bottomPos - topPos) + "px";
		}
	}

	_getMirror() {
		if (this._mirror) {
			this._syncMirrorStyles();
			return this._mirror;
		}
		const m = document.createElement('div');
		m.style.cssText = 'position:absolute;visibility:hidden;pointer-events:none;height:auto;overflow:hidden;';
		this.shadowRoot.appendChild(m);
		this._mirror = m;
		this._syncMirrorStyles();
		return m;
	}

	_syncMirrorStyles() {
		const ta = this._textarea;
		const cs = getComputedStyle(ta);
		const m = this._mirror;
		for (const p of [
			'font', 'letterSpacing', 'wordSpacing', 'textIndent',
			'whiteSpace', 'wordWrap', 'overflowWrap', 'wordBreak',
			'tabSize', 'lineHeight',
			'paddingTop', 'paddingRight', 'paddingBottom', 'paddingLeft',
		]) {
			m.style[p] = cs[p];
		}
		// clientWidth = padding + content (excludes scrollbar and border)
		// With border-box and no border, this gives the same content area as textarea
		m.style.boxSizing = 'border-box';
		m.style.width = ta.clientWidth + 'px';
	}

	_parseLineRanges(attr) {
		return attr.split(",").map(part => {
			part = part.trim();
			const dash = part.indexOf("-");
			if (dash === -1) {
				const n = parseInt(part);
				return { start: n, end: n };
			}
			return {
				start: parseInt(part.substring(0, dash)),
				end: parseInt(part.substring(dash + 1)),
			};
		}).filter(r => !isNaN(r.start) && !isNaN(r.end));
	}

	scrollToLine(lineNumber) {
		if (!this._textarea) return;
		const ta = this._textarea;
		const lines = ta.value.split('\n');
		if (lineNumber < 1 || lineNumber > lines.length) return;
		let pos = 0;
		for (let i = 0; i < lineNumber - 1; i++) {
			pos += lines[i].length + 1;
		}
		const lineLen = lines[lineNumber - 1].length;
		ta.focus();
		ta.setSelectionRange(pos, pos + lineLen);
		const totalLines = lines.length;
		const lineHeight = ta.scrollHeight / totalLines;
		const targetScrollTop = (lineNumber - 1) * lineHeight - ta.clientHeight / 2 + lineHeight / 2;
		ta.scrollTop = Math.max(0, targetScrollTop);
	}
}

customElements.define("c-editor", CEditor);
