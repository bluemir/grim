class CEditor extends HTMLElement {
	static get observedAttributes() {
		return ["highlight"];
	}

	#overlay;
	#highlightRanges;
	#textarea;
	#mirror;
	#gutter;
	#editorBody;
	#onScroll;
	#onInput;
	#lastLineCount;
	#originalValueDesc;
	#resizeObserver;

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
				#editor-body {
					display: flex;
					flex: 1;
					min-height: 0;
					position: relative;
				}
				::slotted(textarea) {
					flex: 1;
					min-height: 0;
				}
				#line-gutter {
					box-sizing: border-box;
					overflow: hidden;
					text-align: right;
					padding-right: 0.5em;
					color: var(--gray-500, #999);
					background: var(--gray-100, #f5f5f5);
					border-right: 1px solid var(--gray-300, #ccc);
					user-select: none;
					cursor: default;
				}
				#highlight-overlay {
					position: absolute;
					pointer-events: none;
					overflow: hidden;
					z-index: 1;
				}
			</style>
			<div id="editor-body">
				<div id="line-gutter"></div>
				<slot></slot>
			</div>
			<div id="highlight-overlay"></div>
		`;
		this.#overlay = this.shadowRoot.getElementById("highlight-overlay");
		this.#gutter = this.shadowRoot.getElementById("line-gutter");
		this.#editorBody = this.shadowRoot.getElementById("editor-body");
		this.#highlightRanges = null;
		this.#textarea = null;
		this.#mirror = null;
		this.#lastLineCount = 0;
		this.#originalValueDesc = null;
		this.#resizeObserver = new ResizeObserver(() => {
			this.#renderLineNumbers();
			this.#syncOverlayInset();
		});
		this.#onScroll = () => {
			this.#updateHighlightPositions();
			this.#syncGutterScroll();
		};
		this.#onInput = () => this.#renderLineNumbers();
	}

	connectedCallback() {
		const slot = this.shadowRoot.querySelector("slot");
		slot.addEventListener("slotchange", () => this.#bindTextarea());
		this.#bindTextarea();
	}

	disconnectedCallback() {
		if (this.#textarea) {
			this.#textarea.removeEventListener("scroll", this.#onScroll);
			this.#textarea.removeEventListener("input", this.#onInput);
			this.#resizeObserver.unobserve(this.#textarea);
			this.#unhookValueSetter();
			this.#textarea = null;
		}
	}

	attributeChangedCallback(name, oldVal, newVal) {
		if (name === "highlight") {
			this.#renderHighlights();
		}
	}

	#bindTextarea() {
		if (this.#textarea) {
			this.#textarea.removeEventListener("scroll", this.#onScroll);
			this.#textarea.removeEventListener("input", this.#onInput);
			this.#resizeObserver.unobserve(this.#textarea);
			this.#unhookValueSetter();
		}
		this.#textarea = this.querySelector("textarea");
		if (!this.#textarea) return;

		const cs = getComputedStyle(this.#textarea);
		if (cs.lineHeight === "normal") {
			this.#textarea.style.lineHeight = "1.2em";
		}

		this.#textarea.addEventListener("scroll", this.#onScroll);
		this.#textarea.addEventListener("input", this.#onInput);
		this.#resizeObserver.observe(this.#textarea);
		this.#hookValueSetter();
		this.#renderLineNumbers();
		this.#syncOverlayInset();
		this.#renderHighlights();
	}

	#hookValueSetter() {
		const ta = this.#textarea;
		const proto = Object.getPrototypeOf(ta);
		const desc = Object.getOwnPropertyDescriptor(proto, 'value')
			|| Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value');
		if (!desc) return;
		this.#originalValueDesc = desc;
		const self = this;
		Object.defineProperty(ta, 'value', {
			get() { return desc.get.call(this); },
			set(v) {
				desc.set.call(this, v);
				self.#renderLineNumbers();
			},
			configurable: true,
		});
	}

	#unhookValueSetter() {
		if (!this.#textarea || !this.#originalValueDesc) return;
		delete this.#textarea.value;
		this.#originalValueDesc = null;
	}

	refresh() {
		this.#renderLineNumbers();
	}

	#syncOverlayInset() {
		if (!this.#textarea) return;
		const cs = getComputedStyle(this.#textarea);
		const gutterWidth = this.#gutter.offsetWidth;
		this.#overlay.style.top = cs.borderTopWidth;
		this.#overlay.style.bottom = cs.borderBottomWidth;
		this.#overlay.style.left = (gutterWidth + parseFloat(cs.borderLeftWidth)) + "px";
		this.#overlay.style.right = cs.borderRightWidth;
	}

	#renderHighlights() {
		const attr = this.getAttribute("highlight");
		if (!attr) {
			this.#overlay.innerHTML = "";
			this.#highlightRanges = null;
			return;
		}

		this.#highlightRanges = this.#parseLineRanges(attr);
		this.#overlay.innerHTML = "";
		for (const range of this.#highlightRanges) {
			const bar = document.createElement("div");
			bar.style.cssText =
				"position:absolute;left:0;right:0;" +
				"pointer-events:none;background:rgba(74,144,226,0.15);";
			this.#overlay.appendChild(bar);
		}
		this.#updateHighlightPositions();
	}

	#updateHighlightPositions() {
		if (!this.#textarea || !this.#highlightRanges) return;
		const ta = this.#textarea;
		const bars = this.#overlay.children;
		const lines = ta.value.split('\n');
		const paddingBottom = parseFloat(getComputedStyle(ta).paddingBottom);

		const mirror = this.#getMirror();
		for (let i = 0; i < this.#highlightRanges.length; i++) {
			const { start, end } = this.#highlightRanges[i];
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

	#getMirror() {
		if (this.#mirror) {
			this.#syncMirrorStyles();
			return this.#mirror;
		}
		const m = document.createElement('div');
		m.style.cssText = 'position:absolute;visibility:hidden;pointer-events:none;height:auto;overflow:hidden;';
		this.shadowRoot.appendChild(m);
		this.#mirror = m;
		this.#syncMirrorStyles();
		return m;
	}

	#syncMirrorStyles() {
		const ta = this.#textarea;
		const cs = getComputedStyle(ta);
		const m = this.#mirror;
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

	#renderLineNumbers() {
		if (!this.#textarea) return;
		const ta = this.#textarea;
		const lines = ta.value.split('\n');
		const lineCount = lines.length;

		// Update gutter width based on digit count
		const digits = String(lineCount).length;
		this.#gutter.style.width = (digits + 1) + "ch";

		// Sync font/lineHeight/padding from textarea
		const cs = getComputedStyle(ta);
		this.#gutter.style.font = cs.font;
		this.#gutter.style.lineHeight = cs.lineHeight;
		this.#gutter.style.paddingTop = cs.paddingTop;
		this.#gutter.style.paddingBottom = cs.paddingBottom;

		// Measure per-line heights: render all lines into mirror as divs, one reflow
		const mirror = this.#getMirror();
		mirror.innerHTML = '';
		const mirrorLines = [];
		for (let i = 0; i < lineCount; i++) {
			const d = document.createElement('div');
			d.textContent = lines[i] || '\u200b';
			mirror.appendChild(d);
			mirrorLines.push(d);
		}

		// Single reflow then batch-read all heights
		const frag = document.createDocumentFragment();
		for (let i = 0; i < lineCount; i++) {
			const div = document.createElement('div');
			div.textContent = i + 1;
			div.style.height = mirrorLines[i].offsetHeight + 'px';
			div.style.overflow = 'hidden';
			frag.appendChild(div);
		}

		this.#gutter.innerHTML = '';
		this.#gutter.appendChild(frag);
		this.#lastLineCount = lineCount;

		// Re-sync overlay inset since gutter width may have changed
		this.#syncOverlayInset();
		this.#updateHighlightPositions();
	}

	#syncGutterScroll() {
		if (!this.#textarea) return;
		this.#gutter.scrollTop = this.#textarea.scrollTop;
	}

	#parseLineRanges(attr) {
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
		if (!this.#textarea) return;
		const ta = this.#textarea;
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
