import { Cloud, FolderKanban, PanelsTopLeft } from 'lucide-react';
import { Navigate, Route, Routes } from 'react-router';

import { useTheme } from './hooks/use-theme';
import { NotFoundPage } from './pages/errors/not-found';
import { ForgesPage } from './pages/forges';
import { Layout } from './pages/layout';
import { MdPage } from './pages/md';
import { ProjectsPage } from './pages/projects';
import { SettingsPage } from './pages/settings';
import { WorkbenchPage } from './pages/workbench';
import type { NavItem } from './pages/layout';

const navItems: NavItem[] = [
  { title: 'Projects', url: '/projects', icon: FolderKanban },
  { title: 'Forges', url: '/forges', icon: Cloud },
  { title: '工作台', url: '/workbench', icon: PanelsTopLeft },
];

function App() {
  // 挂在 App 根上（而非 Layout），保证壳外的独立路由也有 Cmd+D 切主题
  useTheme();

  return (
    <Routes>
      {/* Layout 下的路由 */}
      <Route path="/" element={<Layout navItems={navItems} />}>
        <Route index element={<Navigate to="/projects" replace />} />
        <Route path="projects" element={<ProjectsPage />} />
        <Route path="forges" element={<ForgesPage />} />
        <Route path="workbench" element={<WorkbenchPage />} />
        <Route path="settings" element={<SettingsPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
      {/* 独立的非 Layout 路由 */}
      <Route path="/md" element={<MdPage />} />
    </Routes>
  );
}

export default App;
