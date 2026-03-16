import * as $ from "bm.js/bm.module.js";

function getIndentCharacter(attr) {
    switch(attr) {
        case "2space":
            return "  ";
        case "4space":
            return "    ";
        case "tab":
        default:
            return "\t";
    }
}

class EnhancedTextarea extends HTMLTextAreaElement{
	static get observedAttributes() {
		return ["line-highlight"];
	}
	constructor() {
		super()
	}
	attributeChangedCallback(name, oldVal, newVal) {
		if (name === "line-highlight") {
			this._renderHighlights();
		}
	}
	onConnected()  {
		if(this.hasAttribute("auto-resize")) {
			this.enableAutoResize();
		}

		if(this.hasAttribute("indent")) {
			this.enableIndent();
		}

		if(this.hasAttribute("submit-shortcut")) {
			this.enableSubmitShortcut();
		}

		this._setupHighlightInfra();
	}
	enableAutoResize() {
		if (CSS.supports("field-sizing", "content")) {
			this.style.fieldSizing = "content"
			return
		}

		// try old fashioned way.
		this.style.height = `${this.scrollHeight+2}px`;
		this.on("input", evt => {
			// resize textarea
			let $textarea = evt.target;
			$textarea.style.height = `auto`; // it's magic, shrink area to fit contents
			$textarea.style.height = `${$textarea.scrollHeight+2}px`;
		})
	}
	enableIndent() {
		this.on("keydown", this.#indent);

		if(this.hasAttribute("tab-size")) {
			this.style.tabSize = `${this.attr("tab-size")}ch`
		}
	}
	#indent(evt) {
		switch(evt.code) {
			case "Tab":
				evt.preventDefault();
				let $textarea = evt.target;
				let start = $textarea.selectionStart;
				let end = $textarea.selectionEnd;
				let data = $textarea.value;
				let indent = getIndentCharacter($textarea.attr("indent"))

				if (evt.shiftKey) {
					// un-tab

					let n = data.substring(0, start).lastIndexOf("\n")+1;

					let sections = [data.substring(0, n), data.substring(n, end), data.substring(end)];
					sections[1] = sections[1].split('\n').map(line => line.startsWith(indent)?line.substring(indent.length): line).join('\n');

					$textarea.value = sections.join("");

					$textarea.selectionStart = start > 0 ? start-1: 0;
					$textarea.selectionEnd   = sections[0].length + sections[1].length;
				} else {
					// tab
					// if (end-start > 0 ) { }// mean selection is not empty

					let n = data.substring(0, start).lastIndexOf("\n")+1;
					let sections = [data.substring(0, n), data.substring(n, end), data.substring(end)];

					sections[1] = sections[1].split('\n').map(line => indent + line).join('\n');

					$textarea.value = sections.join("");

					$textarea.selectionStart = start + indent.length;
					$textarea.selectionEnd   = sections[0].length + sections[1].length;
				}
				return;
			case "Enter":
				{
					let $textarea = evt.target;
					let start = $textarea.selectionStart;
					let end = $textarea.selectionEnd;
					let data = $textarea.value;

					if (end - start > 0) {
						return; // skip. it has selection
					}

					evt.preventDefault();

					// insert newline & indent
					let n = data.substring(0, start).lastIndexOf("\n")+1;

					let lastLine = data.substring(n, start);

					let indent = lastLine; // if empty line, use whole line.
					let matched = lastLine.match(/[^\s]/);
					if (matched){
						indent = data.substring(n, n + lastLine.match(/[^\s]/).index);
					}

					let arr = [data.substring(0, start), "\n", indent, data.substring(start)];

					$textarea.value = arr.join("");
					$textarea.selectionStart = $textarea.selectionEnd = arr[0].length + arr[1].length + arr[2].length;

					return;
				}
			default:
				//console.log(evt);
		}
	}
	_setupHighlightInfra() {
		if (this._highlightContainer) return;
		if (!this.parentElement) return;

		const cs = getComputedStyle(this);
		if (cs.lineHeight === "normal") {
			this.style.lineHeight = "1.2em";
		}

		const wrapper = document.createElement("div");
		wrapper.style.cssText = "position:relative;flex:1;min-height:0;display:flex;flex-direction:column;";
		this.parentElement.insertBefore(wrapper, this);
		wrapper.appendChild(this);

		const csAfter = getComputedStyle(this);
		const container = document.createElement("div");
		container.style.cssText =
			"position:absolute;pointer-events:none;overflow:hidden;z-index:1;" +
			`top:${parseFloat(csAfter.borderTopWidth)}px;` +
			`bottom:${parseFloat(csAfter.borderBottomWidth)}px;` +
			`left:${parseFloat(csAfter.borderLeftWidth)}px;` +
			`right:${parseFloat(csAfter.borderRightWidth)}px;`;
		wrapper.appendChild(container);
		this._highlightContainer = container;

		this.addEventListener("scroll", () => this._updateHighlightPositions());
	}
	_renderHighlights() {
		if (!this._highlightContainer) return;

		const attr = this.getAttribute("line-highlight");
		if (!attr) {
			this._highlightContainer.innerHTML = "";
			this._highlightRanges = null;
			return;
		}

		this._highlightRanges = this._parseLineRanges(attr);
		this._highlightContainer.innerHTML = "";
		for (const range of this._highlightRanges) {
			const bar = document.createElement("div");
			bar.style.cssText =
				"position:absolute;left:0;right:0;" +
				"pointer-events:none;background:rgba(74,144,226,0.15);";
			this._highlightContainer.appendChild(bar);
		}
		this._updateHighlightPositions();
	}
	_updateHighlightPositions() {
		if (!this._highlightContainer || !this._highlightRanges) return;
		const style = getComputedStyle(this);
		const lineHeight = parseFloat(style.lineHeight);
		const paddingTop = parseFloat(style.paddingTop);
		const bars = this._highlightContainer.children;

		for (let i = 0; i < this._highlightRanges.length; i++) {
			const { start, end } = this._highlightRanges[i];
			const top = paddingTop + (start - 1) * lineHeight - this.scrollTop;
			const height = (end - start + 1) * lineHeight;
			bars[i].style.top = top + "px";
			bars[i].style.height = height + "px";
		}
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
		const lines = this.value.split('\n');
		if (lineNumber < 1 || lineNumber > lines.length) return;
		let pos = 0;
		for (let i = 0; i < lineNumber - 1; i++) {
			pos += lines[i].length + 1;
		}
		const lineLen = lines[lineNumber - 1].length;
		this.focus();
		this.setSelectionRange(pos, pos + lineLen);
		const totalLines = lines.length;
		const lineHeight = this.scrollHeight / totalLines;
		const targetScrollTop = (lineNumber - 1) * lineHeight - this.clientHeight / 2 + lineHeight / 2;
		this.scrollTop = Math.max(0, targetScrollTop);
	}
	enableSubmitShortcut() {
		this.on("keydown", this.#submitShortcut)
	}
	async #submitShortcut(evt) {
		if (!(evt.code == "KeyS" && (evt.ctrlKey || evt.metaKey))) {
			return // just skip
		}
		evt.preventDefault();

		let $form = this.closest("form");
		if (!$form) {
			return // no form, just prevent default browser save
		}

		let data = new FormData($form);
		let res = await $.request($form.attr("method")||$form.method, $form.action||location.pathname, {body: data});
		// TODO show message
	}
}
customElements.define("enhanced-textarea", EnhancedTextarea, {extends: "textarea"});

