import { Controller } from "@hotwired/stimulus";

export default class extends Controller {
	connect() {
		this.updateActiveListener = (event) => {
			if (event.path === undefined) {
				// This can happen if the controller was connected to an element from a Turbo Streams
				// refresh action instead of a Turbo Drive page navigation.
				return;
			}

			console.log(event.path);
			const location = event.path[2].location.href;
			if (this.element === undefined) {
				return;
			}

			if (location.startsWith(this.element.href)) {
				this.element.classList.add("is-active");
			} else {
				this.element.classList.remove("is-active");
			}
		};

		document.addEventListener("turbo:render", this.updateActiveListener);
	}
	disconnect() {
		if (this.updateActiveListener === undefined) {
			return;
		}

		document.removeEventListener("turbo:render", this.updateActiveListener);
	}

	updateActive;
}
