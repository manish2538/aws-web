import { Routes, Route, Link, NavLink } from 'react-router-dom';
import DashboardPage from './pages/DashboardPage';
import ServiceDetailPage from './pages/ServiceDetailPage';
import ResourcesOverviewPage from './pages/ResourcesOverviewPage';
import CommandRunnerPage from './pages/CommandRunnerPage';
import ProfileBar from './components/ProfileBar';
import CurrencySelector from './components/CurrencySelector';

function App() {
  return (
    <div className="app-root">
      <header className="app-header">
        <div className="app-header-inner">
          <div className="flex items-center">
            <Link to="/" className="app-logo">
              AWS Local Dashboard
            </Link>
            <nav className="app-nav">
              <NavLink
                to="/"
                end
                className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}
              >
                Cost Explorer
              </NavLink>
              <NavLink
                to="/resources"
                className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}
              >
                Resources
              </NavLink>
              <NavLink
                to="/commands"
                className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}
              >
                CLI Runner
              </NavLink>
            </nav>
          </div>
          <div className="flex items-center gap-md">
            <CurrencySelector />
            <div className="toolbar-divider" style={{ height: 20 }} />
            <ProfileBar />
          </div>
        </div>
      </header>
      <main className="app-main">
        <Routes>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/services/:serviceKey" element={<ServiceDetailPage />} />
          <Route path="/resources" element={<ResourcesOverviewPage />} />
          <Route path="/commands" element={<CommandRunnerPage />} />
        </Routes>
      </main>
    </div>
  );
}

export default App;
