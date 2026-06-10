import type { processmonitor } from "../../wailsjs/go/models";

interface KillConfirmDialogProps {
  server: processmonitor.Server;
  onCancel: () => void;
  onConfirm: () => void;
}

export default function KillConfirmDialog({
  server,
  onCancel,
  onConfirm,
}: KillConfirmDialogProps) {
  return (
    <div
      className="pk-kill-overlay"
      onClick={onCancel}
      role="dialog"
      aria-modal="true"
    >
      <div
        className="pk-kill-dialog"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="pk-kill-dialog-msg">
          Kill{" "}
          <span className="pk-kill-dialog-name">
            {server.processName}
          </span>{" "}
          on :{server.port}?
        </div>
        <div className="pk-kill-dialog-actions">
          <button
            className="pk-kill-btn pk-kill-btn-cancel"
            onClick={onCancel}
          >
            Cancel
          </button>
          <button
            className="pk-kill-btn pk-kill-btn-confirm"
            onClick={onConfirm}
          >
            Kill
          </button>
        </div>
      </div>
    </div>
  );
}
