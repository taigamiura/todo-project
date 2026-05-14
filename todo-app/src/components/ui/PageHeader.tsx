import { ReactNode } from "react";

type PageHeaderProps = {
    eyebrow: string;
    title: string;
    description: string;
    actions?: ReactNode;
};

export function PageHeader({ eyebrow, title, description, actions }: PageHeaderProps) {
    return (
        <header className="flex flex-col gap-5 rounded-[32px] bg-slate-950 px-6 py-8 text-white shadow-[0_40px_120px_rgba(15,23,42,0.3)] sm:px-8 lg:flex-row lg:items-end lg:justify-between">
            <div>
                <p className="text-sm uppercase tracking-[0.28em] text-sky-300">{eyebrow}</p>
                <h1 className="mt-4 text-3xl font-semibold leading-tight sm:text-4xl">{title}</h1>
                <p className="mt-4 max-w-2xl text-sm leading-7 text-slate-300 sm:text-base">{description}</p>
            </div>
            {actions ? <div className="shrink-0">{actions}</div> : null}
        </header>
    );
}