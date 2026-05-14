type EmptyStateProps = {
    title: string;
    description: string;
};

export function EmptyState({ title, description }: EmptyStateProps) {
    return (
        <div className="rounded-[24px] border border-dashed border-slate-300 bg-slate-50 px-6 py-10 text-center">
            <h3 className="text-xl font-semibold text-slate-950">{title}</h3>
            <p className="mx-auto mt-3 max-w-md text-sm leading-7 text-slate-600">{description}</p>
        </div>
    );
}