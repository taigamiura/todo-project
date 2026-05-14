import { ComponentPropsWithoutRef } from "react";

type ButtonVariant = "primary" | "secondary" | "danger";

type ButtonProps = ComponentPropsWithoutRef<"button"> & {
    variant?: ButtonVariant;
    fullWidth?: boolean;
};

const variantClassName: Record<ButtonVariant, string> = {
    primary: "bg-slate-950 text-white hover:bg-slate-800",
    secondary: "border border-slate-200 bg-white text-slate-700 hover:border-slate-300 hover:text-slate-950",
    danger: "bg-rose-600 text-white hover:bg-rose-700",
};

export function Button({
    className = "",
    variant = "primary",
    fullWidth = false,
    ...props
}: ButtonProps) {
    return (
        <button
            className={[
                "inline-flex items-center justify-center rounded-full px-5 py-3 text-sm font-semibold transition disabled:cursor-not-allowed disabled:opacity-60",
                variantClassName[variant],
                fullWidth ? "w-full" : "",
                className,
            ].join(" ")}
            {...props}
        />
    );
}