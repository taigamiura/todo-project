import { ComponentPropsWithoutRef } from "react";

type CardProps = ComponentPropsWithoutRef<"section">;

export function Card({ className = "", ...props }: CardProps) {
    return (
        <section
            className={[
                "rounded-[28px] border border-black/5 bg-white p-6 shadow-[0_24px_80px_rgba(15,23,42,0.08)] sm:p-8",
                className,
            ].join(" ")}
            {...props}
        />
    );
}