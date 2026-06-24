import {
	basicSetup, EditorView, EditorState, keymap,
	indentWithTab, indentUnit,
	StateField, StateEffect, Decoration,
	lintGutter, setDiagnostics,
} from "codemirror";

// --- highlight line decoration ---

const setHighlightEffect = StateEffect.define();

const highlightField = StateField.define({
	create() { return Decoration.none; },
	update(decorations, tr) {
		for (const e of tr.effects) {
			if (e.is(setHighlightEffect)) {
				if (e.value.length === 0) return Decoration.none;
				const marks = [];
				for (const { start, end } of e.value) {
					for (let line = start; line <= end; line++) {
						if (line >= 1 && line <= tr.state.doc.lines) {
							marks.push(highlightLineDeco.range(tr.state.doc.line(line).from));
						}
					}
				}
				return Decoration.set(marks);
			}
		}
		return decorations.map(tr.changes);
	},
	provide: f => EditorView.decorations.from(f),
});

const highlightLineDeco = Decoration.line({
	class: "cm-highlighted-line",
});

// --- custom element ---

class CEditor extends HTMLElement {
	static get observedAttributes() {
		return ["highlight"];
	}

	#view;
	#container;

	constructor() {
		super();
		this.attachShadow({ mode: "open" });
		this.shadowRoot.innerHTML = `
			<style>
				:host {
					display: flex;
					flex: 1;
					min-height: 0;
				}
				#cm-container {
					flex: 1;
					min-height: 0;
					overflow: auto;
				}
				#cm-container .cm-editor {
					height: 100%;
				}
				#cm-container .cm-scroller {
					overflow: auto;
				}
				#cm-container .cm-highlighted-line {
					background: rgba(74, 144, 226, 0.15);
				}
			</style>
			<div id="cm-container"></div>
		`;
		this.#container = this.shadowRoot.getElementById("cm-container");
		this.#view = null;
	}

	connectedCallback() {
		if (this.#view) return;

		const self = this;

		this.#view = new EditorView({
			root: this.shadowRoot,
			parent: this.#container,
			state: EditorState.create({
				doc: "",
				extensions: [
					basicSetup,
					EditorView.lineWrapping,
					indentUnit.of("\t"),
					EditorState.tabSize.of(4),
					keymap.of([
						indentWithTab,
						{
							key: "Mod-s",
							run() {
								const form = self.closest("form");
								if (form) form.requestSubmit();
								return true;
							},
						},
					]),
					highlightField,
					lintGutter(),
					EditorView.updateListener.of((update) => {
						if (update.docChanged) {
							self.dispatchEvent(new Event("input", { bubbles: true }));
						}
					}),
				],
			}),
		});
	}

	disconnectedCallback() {
		if (this.#view) {
			this.#view.destroy();
			this.#view = null;
		}
	}

	attributeChangedCallback(name, oldVal, newVal) {
		if (name === "highlight") {
			this.#applyHighlight();
		}
	}

	get value() {
		if (!this.#view) return "";
		return this.#view.state.doc.toString();
	}

	set value(text) {
		if (!this.#view) return;
		// Remember cursor line before replacing
		const cursorLine = this.#view.state.doc.lineAt(
			this.#view.state.selection.main.head
		).number;
		this.#view.dispatch({
			changes: { from: 0, to: this.#view.state.doc.length, insert: text },
		});
		// Restore cursor to the same line (clamped to new doc length)
		this.#scrollToLineAfterUpdate(cursorLine);
	}

	#scrollToLineAfterUpdate(lineNumber) {
		// Defer so CodeMirror finishes its internal layout/scroll update first
		requestAnimationFrame(() => {
			if (!this.#view) return;
			const doc = this.#view.state.doc;
			const targetLine = Math.min(lineNumber, doc.lines);
			const pos = doc.line(targetLine).from;
			this.#view.dispatch({
				selection: { anchor: pos },
				effects: EditorView.scrollIntoView(pos, { y: "center" }),
			});
		});
	}

	refresh() {
		if (this.#view) this.#view.requestMeasure();
	}

	scrollToLine(lineNumber) {
		if (!this.#view) return;
		const doc = this.#view.state.doc;
		if (lineNumber < 1 || lineNumber > doc.lines) return;
		const line = doc.line(lineNumber);
		this.#view.dispatch({
			selection: { anchor: line.from },
			effects: EditorView.scrollIntoView(line.from, { y: "center" }),
		});
		this.#view.focus();
	}

	// Show parse diagnostics as gutter markers + inline underlines.
	// diags: array of { line, col, message } (1-based line/col).
	// Pass an empty array to clear all diagnostics.
	setDiagnostics(diags) {
		if (!this.#view) return;
		const doc = this.#view.state.doc;
		const cmDiags = (diags || []).map(d => {
			const lineNo = Math.min(Math.max(d.line || 1, 1), doc.lines);
			const line = doc.line(lineNo);
			let from = line.from + Math.max(0, (d.col || 1) - 1);
			let to = line.to;
			// Positionless (col 0) or end-of-line errors: underline the whole line.
			if (from >= to) {
				from = line.from;
				to = line.to;
			}
			return { from, to, severity: "error", message: d.message };
		});
		this.#view.dispatch(setDiagnostics(this.#view.state, cmDiags));
	}

	#applyHighlight() {
		if (!this.#view) return;
		const attr = this.getAttribute("highlight");
		if (!attr) {
			this.#view.dispatch({ effects: setHighlightEffect.of([]) });
			return;
		}
		const ranges = this.#parseLineRanges(attr);
		this.#view.dispatch({ effects: setHighlightEffect.of(ranges) });
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
}

customElements.define("c-editor", CEditor);
