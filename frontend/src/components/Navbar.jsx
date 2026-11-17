import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { useLocale } from '../lib/locale';
import LanguageSwitch from './LanguageSwitch';

export default function Navbar() {
  const { user, logout } = useAuth();
  const { tr } = useLocale();
  const navigate = useNavigate();
  const premium = user?.subscription_tier === 'premium';
  return <nav className="main-nav">
    <Link to="/" className="brand-mark" aria-label="Learnthor"><span aria-hidden="true">★</span> LEARN<span>THOR</span><small>日本語</small></Link>
    <div className="nav-actions">
      <div className="nav-routes"><Link to="/learn">{tr('Learn')}</Link><Link to="/kanji">{tr('Kanji')}</Link></div>
      <LanguageSwitch />
      {premium ? <span className="premium-tag">PREMIUM</span> : <Link className="premium-tag" to="/upgrade">{tr('👑 Upgrade')}</Link>}
      <span className="nav-username">{user?.display_name || user?.username}</span>
      <button className="nav-logout" onClick={() => { logout(); navigate('/login'); }}>{tr('Log out')} ↗</button>
    </div>
  </nav>;
}
