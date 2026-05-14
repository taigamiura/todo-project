import { ReactNode } from "react";

type FieldProps = {
    label: string;
    htmlFor: string;
    error?: string;
    children: ReactNode;
};

export function Field({ label, htmlFor, error, children }: FieldProps) {
    return (
        <label className="block" htmlFor={htmlFor}>
            <span className="mb-2 block text-sm font-medium text-slate-800">{label}</span>
            {children}
            {error ? <span className="mt-2 block text-sm text-rose-600">{error}</span> : null}
        </label>
    );
}