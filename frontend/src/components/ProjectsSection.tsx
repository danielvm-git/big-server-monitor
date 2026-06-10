import { Folder } from "lucide-react";
import type { processmonitor } from "../../wailsjs/go/models";
import ServerRow from "./ServerRow";

interface ProjectsSectionProps {
  projectName: string;
  projectDir: string;
  servers: processmonitor.Server[];
  onKill: (pid: number) => void;
}

export default function ProjectsSection({
  projectName,
  projectDir,
  servers,
  onKill,
}: ProjectsSectionProps) {
  return (
    <div className="pk-project-section">
      <div className="pk-project-header">
        <Folder size={12} strokeWidth={2} className="pk-project-header-icon" />
        <span>{projectName}</span>
        {projectDir && (
          <span className="pk-project-header-dir">{projectDir}</span>
        )}
      </div>
      {servers.map((server) => (
        <ServerRow key={server.port} server={server} onKill={onKill} />
      ))}
    </div>
  );
}
