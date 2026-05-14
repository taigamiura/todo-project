import { type ReactElement } from "react";
import { render } from "@testing-library/react";

export function renderElement(element: ReactElement) {
    return render(element);
}

export async function renderAsyncComponent<T>(component: () => Promise<T>) {
    const result = await component();
    return render(result as ReactElement);
}