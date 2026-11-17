import LanguageSwitch from '../components/LanguageSwitch';
import { useLocale } from '../lib/locale';
import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import { useAuth } from '../context/AuthContext';
import { friendlyError } from '../lib/errors';

export default function RegisterPage() {
  const { tr, language } = useLocale();
  const { register } = useAuth();
  const navigate = useNavigate();
  const [form, setForm] = useState({ username: '', email: '', password: '' });
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e) {
    e.preventDefault();
    setError('');
    if (form.password.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }
    setLoading(true);
    try {
      await register(form.username, form.email, form.password);
      navigate('/');
    } catch (err) {
      setError(friendlyError(err, language));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="auth-stage">
      <div className="auth-language"><LanguageSwitch /></div>
      <div className="auth-poster" aria-hidden="true"><span>LEARN<br/>THOR.</span><b>学</b><small>TAKE YOUR NEXT STEP.</small></div>
      <motion.div
        initial={{ opacity: 0, y: 20, scale: 0.97 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        transition={{ type: 'spring', stiffness: 260, damping: 22 }}
        className="auth-card"
      >
        <div className="text-center mb-6">
          <div className="text-4xl mb-2">✦</div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-slate-100 mb-1">{tr("Create account")}</h1>
          <p className="text-gray-500 dark:text-slate-400 text-sm">{tr("Start your Japanese learning journey")}</p>
        </div>

        {error && (
          <div className="mb-4 p-3 rounded-xl bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-300 text-sm text-center">
            {typeof error === 'string' ? tr(error) : friendlyError(error, language)}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-slate-300 mb-1">{tr("Username")}</label>
            <input
              type="text"
              required
              minLength={3}
              value={form.username}
              onChange={(e) => setForm({ ...form, username: e.target.value })}
              className="w-full px-3 py-2.5 border border-gray-300 dark:border-slate-600 dark:bg-slate-700 rounded-xl text-sm text-gray-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition-shadow"
              placeholder={tr("taro")}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-slate-300 mb-1">{tr("Email")}</label>
            <input
              type="email"
              required
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
              className="w-full px-3 py-2.5 border border-gray-300 dark:border-slate-600 dark:bg-slate-700 rounded-xl text-sm text-gray-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition-shadow"
              placeholder={tr("you@example.com")}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-slate-300 mb-1">{tr("Password")}</label>
            <input
              type="password"
              required
              minLength={8}
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
              className="w-full px-3 py-2.5 border border-gray-300 dark:border-slate-600 dark:bg-slate-700 rounded-xl text-sm text-gray-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition-shadow"
              placeholder={tr("min 8 characters")}
            />
          </div>
          <motion.button
            type="submit"
            disabled={loading}
            whileTap={{ scale: 0.96 }}
            className="w-full bg-indigo-600 hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-600 disabled:opacity-50 text-white font-semibold py-3 rounded-xl text-sm transition-colors shadow-soft"
          >
            {loading ? tr("Creating account…") : tr("Create account →")}
          </motion.button>
        </form>

        <p className="mt-6 text-center text-sm text-gray-500 dark:text-slate-400">{tr("Already have an account?")}{' '}
          <Link
            to="/login"
            className="text-indigo-600 dark:text-indigo-400 hover:underline font-semibold"
          >{tr("Log in")}</Link>
        </p>
      </motion.div>
    </div>
  );
}