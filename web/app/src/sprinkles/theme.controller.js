import { Controller } from "@hotwired/stimulus";

import {
	invert,
	loadThemeSetting,
	setTheme,
	storeThemeSetting,
} from "./util/theme-toggle";

export default class extends Controller {
	connect() {
		this.element.classList.remove("is-hidden");
	}
	toggle() {
		const theme = invert(loadThemeSetting());
		setTheme(theme);
		storeThemeSetting(theme);
	}
}
