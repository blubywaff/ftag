import { browser } from '$app/environment';

export interface Settings {
	defaultExcludes: string;
	defaultTagView: 'hide' | 'show' | 'edit';
}

const KEY = 'ftag_settings';

const defaultSettings: Settings = {
	defaultExcludes: '',
	defaultTagView: 'show'
};

export const options = {
	tagView: ['hide', 'show', 'edit']
};

export const settings: Settings = $state(JSON.parse(JSON.stringify(defaultSettings)));

export function showTags(): boolean {
	return settings.defaultTagView !== 'hide';
}

export function showTagEdit(): boolean {
	return settings.defaultTagView === 'edit';
}

export function read(): void {
	if (!browser) {
		return;
	}
	const def: string = JSON.stringify(defaultSettings);
	const saved: Settings = JSON.parse(localStorage.getItem(KEY) || def);
	settings.defaultTagView = saved.defaultTagView;
	settings.defaultExcludes = saved.defaultExcludes;
}

export function write(): void {
	if (!browser) {
		return;
	}
	localStorage.setItem(KEY, JSON.stringify(settings));
}
