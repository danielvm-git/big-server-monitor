import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import type { processmonitor } from "../../wailsjs/go/models";

interface ServerRowExpandedProps {
  server: processmonitor.Server;
}

export default function ServerRowExpanded({
  server,
}: ServerRowExpandedProps) {
  const [showEnv, setShowEnv] = useState(false);

  return (
    <div className="pk-server-expanded">
      <div className="pk-server-expanded-field">
        <span className="pk-server-expanded-label">PID</span>
        <span className="pk-server-expanded-value">{server.pid}</span>
      </div>
      {server.binaryPath && (
        <div className="pk-server-expanded-field">
          <span className="pk-server-expanded-label">Binary</span>
          <span className="pk-server-expanded-value">
            {server.binaryPath}
          </span>
        </div>
      )}
      {server.memoryMb > 0 && (
        <div className="pk-server-expanded-field">
          <span className="pk-server-expanded-label">Memory</span>
          <span className="pk-server-expanded-value">
            {server.memoryMb.toFixed(1)} MB
          </span>
        </div>
      )}
      {server.runtimeVersion && (
        <div className="pk-server-expanded-field">
          <span className="pk-server-expanded-label">Runtime</span>
          <span className="pk-server-expanded-value">
            {server.runtimeVersion}
          </span>
        </div>
      )}
      {server.localDomain && (
        <div className="pk-server-expanded-field">
          <span className="pk-server-expanded-label">Domain</span>
          <a
            className="pk-server-expanded-link"
            href={`http://${server.localDomain}`}
            target="_blank"
            rel="noreferrer"
          >
            {server.localDomain}
          </a>
        </div>
      )}
      {server.tunnelURL && (
        <div className="pk-server-expanded-field">
          <span className="pk-server-expanded-label">Tunnel</span>
          <a
            className="pk-server-expanded-pill"
            href={server.tunnelURL}
            target="_blank"
            rel="noreferrer"
          >
            {server.tunnelURL}
          </a>
        </div>
      )}
      {server.envSnapshot && server.envSnapshot.length > 0 && (
        <div>
          <button
            className="pk-server-expanded-env-toggle"
            onClick={() => setShowEnv((prev) => !prev)}
          >
            {showEnv ? (
              <ChevronDown size={12} strokeWidth={2} />
            ) : (
              <ChevronRight size={12} strokeWidth={2} />
            )}
            Env ({server.envSnapshot.length})
          </button>
          {showEnv && (
            <div className="pk-server-expanded-env-list">
              {server.envSnapshot.map((env) => (
                <div
                  key={env.key}
                  className="pk-server-expanded-env-item"
                >
                  <span className="pk-server-expanded-env-key">
                    {env.key}
                  </span>
                  <span>
                    {env.visible ? env.value : "***"}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
