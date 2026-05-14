import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { Field } from "@/components/ui/Field";
import { Input } from "@/components/ui/Input";
import { PageHeader } from "@/components/ui/PageHeader";

describe("ui components", () => {
    it("renders button variants", () => {
        const { rerender } = render(<Button fullWidth>ok</Button>);
        expect(screen.getByRole("button")).toHaveClass("w-full");
        rerender(<Button variant="secondary">secondary</Button>);
        expect(screen.getByRole("button")).toHaveClass("border");
        rerender(<Button variant="danger">danger</Button>);
        expect(screen.getByRole("button")).toHaveClass("bg-rose-600");
    });

    it("renders card, field, input, empty state and page header", () => {
        render(
            <div>
                <Card data-testid="card">body</Card>
                <Field label="Name" htmlFor="name" error="Required"><Input id="name" /></Field>
                <EmptyState title="No items" description="Add one" />
                <PageHeader eyebrow="Test" title="Header" description="Desc" actions={<span>Action</span>} />
            </div>,
        );

        expect(screen.getByTestId("card")).toHaveTextContent("body");
        expect(screen.getByText("Required")).toBeInTheDocument();
        expect(screen.getByRole("textbox")).toHaveClass("rounded-2xl");
        expect(screen.getByText("No items")).toBeInTheDocument();
        expect(screen.getByText("Action")).toBeInTheDocument();
    });

    it("renders field without error and header without actions", () => {
        render(
            <div>
                <Field label="Email" htmlFor="email"><Input id="email" /></Field>
                <PageHeader eyebrow="Test" title="Header" description="Desc" />
            </div>,
        );
        expect(screen.queryByText("Required")).not.toBeInTheDocument();
    });
});