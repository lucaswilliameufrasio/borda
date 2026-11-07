export function inputValue(event: Event): string {
	const target = event.currentTarget;
	if (target instanceof HTMLInputElement || target instanceof HTMLTextAreaElement) {
		return target.value;
	}
	return '';
}

export function formElement(event: SubmitEvent): HTMLFormElement | undefined {
	const target = event.currentTarget;
	if (target instanceof HTMLFormElement) {
		return target;
	}
	return undefined;
}

export function formString(formData: FormData, name: string): string {
	const value = formData.get(name);
	if (typeof value === 'string') {
		return value;
	}
	return '';
}
