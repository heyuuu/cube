import { Link } from 'react-router';

export function NotFoundPage() {
  return (
    <div className="p-6">
      <h1 className="text-lg font-semibold">404</h1>
      <p className="text-muted-foreground mt-1 text-xs">页面不存在</p>
      <Link to="/projects" className="text-primary mt-4 inline-block text-xs hover:underline">
        回项目列表
      </Link>
    </div>
  );
}
