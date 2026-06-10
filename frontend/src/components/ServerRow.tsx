import { useState } from "react";
import { X } from "lucide-react";
import type { processmonitor } from "../../wailsjs/go/models";
import ServerRowExpanded from "./ServerRowExpanded";
import KillConfirmDialog from "./KillConfirmDialog";

interface ServerRowProps {
  server: processmonitor.Server;
  onKill: (pid: number) => void;
}

export default function ServerRow({ server, onKill }: ServerRowProps) {
  const [expanded, setExpanded] = useState(false);
  const [showKill, setShowKill] = useState(false);

  const dotClass = `pk-server-row-dot ${server.status}`;

  return (
    <>
      <div
        className="pk-server-row"
        onClick={() => setExpanded((prev) => !prev)}
        role="button"
        tabIndex={0}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") setExpanded((prev) => !prev);
        }}
      >
        <span className={dotClass} />
        <div className="pk-server-row-info">
          <span className="pk-server-row-name">{server.processName}</span>
          <span className="pk-server-row-port">:{server.port}</span>
        </div>
        <span className="pk-server-row-uptime">{server.uptimeStr}</span>
        <button
          className="pk-server-row-kill"
          onClick={(e) => {
            e.stopPropagation();
            setShowKill(true);
          }}
          aria-label={`Kill ${server.processName}`}
        >
          <X size={14} strokeWidth={2} />
        </button>
      </div>
      {expanded && <ServerRowExpanded server={server} />}
      {showKill && (
        <KillConfirmDialog
          server={server}
          onCancel={() => setShowKill(false)}
          onConfirm={() => {
            onKill(server.pid);
            setShowKill(false);
          }}
        />
      )}
    </>
  );
}
