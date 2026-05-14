"use client";

import { useState } from "react";
import { Button } from "@/components/ui/Button";

type TodoDetailActionsProps = {
    onDelete: () => Promise<void>;
};

export function TodoDetailActions({ onDelete }: TodoDetailActionsProps) {
    const [isDeleting, setIsDeleting] = useState(false);

    return (
        <div className="space-y-4">
            <h3 className="text-lg font-semibold text-slate-950">削除</h3>
            <p className="text-sm leading-7 text-slate-600">
                削除するとこの Todo は一覧からも消えます。必要なら一覧へ戻る前に内容を確認してください。
            </p>
            <Button
                variant="danger"
                disabled={isDeleting}
                onClick={async () => {
                    setIsDeleting(true);
                    await onDelete();
                }}
            >
                {isDeleting ? "削除中..." : "Todoを削除"}
            </Button>
        </div>
    );
}