import { useLocale } from '../lib/locale';
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import api from '../lib/api';
import { friendlyError } from '../lib/errors';
import { useAuth } from '../context/AuthContext';

const BENEFITS = [
  { icon: '🔓', text: 'Unlock all JLPT N4 & N3 vocabulary modules' },
  { icon: '📅', text: 'Daily Vocab series (Day 29–59)' },
  { icon: '🔄', text: 'Full spaced-repetition review across every level' },
  { icon: '♾️', text: 'One-time payment — lifetime access, no subscription' },
];

// loadSnapJs injects the Midtrans snap.js script once and resolves when ready.
function loadSnapJs(clientKey, env) {
  return new Promise((resolve, reject) => {
    if (window.snap) return resolve(window.snap);
    const script = document.createElement('script');
    script.src =
      env === 'production'
        ? 'https://app.midtrans.com/snap/snap.js'
        : 'https://app.sandbox.midtrans.com/snap/snap.js';
    script.setAttribute('data-client-key', clientKey);
    script.onload = () => resolve(window.snap);
    script.onerror = () => reject(new Error('snap.js failed to load'));
    document.body.appendChild(script);
  });
}

// pollUntilSettled polls the backend until the webhook has flipped the
// payment to settlement (or gives up after ~20s).
async function pollUntilSettled(orderId) {
  for (let i = 0; i < 13; i++) {
    const { data } = await api.get(`/payments/${orderId}`);
    if (data.status === 'settlement') return data;
    await new Promise((r) => setTimeout(r, 1500));
  }
  return null;
}

function formatIDR(amount) {
  return 'Rp ' + amount.toLocaleString('id-ID');
}

export default function UpgradePage() {
  const { tr, language } = useLocale();
  const navigate = useNavigate();
  const { user, refreshUser } = useAuth();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [stage, setStage] = useState('pitch'); // pitch | waiting | success | pending

  const isPremium = user?.subscription_tier === 'premium';

  async function startCheckout() {
    setLoading(true);
    setError('');
    try {
      const { data } = await api.post('/payments/create');
      const snap = await loadSnapJs(data.client_key, data.env);

      snap.pay(data.snap_token, {
        onSuccess: () => finishPayment(data.order_id),
        onPending: () => finishPayment(data.order_id),
        onError: () => {
          setStage('pitch');
          setError(
            language === 'id'
              ? 'Pembayaran gagal. Silakan coba lagi.'
              : 'Payment failed. Please try again.',
          );
        },
        onClose: () => {
          // User closed the popup without paying — only reset if we never
          // started polling (onSuccess/onPending fire first otherwise).
          setStage((s) => (s === 'waiting' ? s : 'pitch'));
        },
      });
    } catch (err) {
      setError(friendlyError(err, language));
    } finally {
      setLoading(false);
    }
  }

  async function finishPayment(orderId) {
    setStage('waiting');
    try {
      const settled = await pollUntilSettled(orderId);
      if (settled) {
        refreshUser({ subscription_tier: 'premium' });
        setStage('success');
      } else {
        setStage('pending');
      }
    } catch {
      setStage('pending');
    }
  }

  if (isPremium || stage === 'success') {
    return (
      <div className="max-w-lg mx-auto px-4 py-16 text-center">
        <div className="text-6xl mb-4">👑</div>
        <h1 className="text-2xl font-bold text-gray-900 mb-2">
          {stage === 'success' ? tr("Welcome to Premium!") : tr("You're Premium!")}
        </h1>
        <p className="text-gray-500 mb-8">{tr(user?.progression_bypass ? 'Learning access granted. Your role remains student.' : 'Premium includes the content. Learning prerequisites still apply.')}</p>
        <button
          onClick={() => navigate('/')}
          className="px-8 py-3 bg-indigo-600 text-white rounded-2xl font-semibold hover:bg-indigo-700 transition-colors"
        >{tr("Start learning →")}</button>
      </div>
    );
  }

  if (stage === 'waiting') {
    return (
      <div className="max-w-lg mx-auto px-4 py-16 text-center">
        <div className="w-10 h-10 mx-auto mb-4 border-4 border-indigo-600 border-t-transparent rounded-full animate-spin" />
        <h1 className="text-xl font-bold text-gray-900 mb-2">{tr("Confirming your payment…")}</h1>
        <p className="text-gray-500 text-sm">{tr("This usually takes a few seconds. Don't close the page.")}</p>
      </div>
    );
  }

  if (stage === 'pending') {
    return (
      <div className="max-w-lg mx-auto px-4 py-16 text-center">
        <div className="text-6xl mb-4">⏳</div>
        <h1 className="text-xl font-bold text-gray-900 mb-2">{tr("Payment is being processed")}</h1>
        <p className="text-gray-500 mb-8">{tr("We're still waiting for confirmation from the payment provider. Your account will upgrade automatically once it completes.")}</p>
        <button
          onClick={() => navigate('/')}
          className="px-8 py-3 border border-gray-200 text-gray-600 rounded-2xl font-medium hover:bg-gray-50 transition-colors"
        >{tr("← Back to home")}</button>
      </div>
    );
  }

  return (
    <div className="max-w-lg mx-auto px-4 py-8">
      <button
        onClick={() => navigate(-1)}
        className="text-sm text-gray-400 hover:text-gray-600 mb-6"
      >{tr("← Back")}</button>

      <div className="bg-gradient-to-br from-indigo-600 to-violet-600 rounded-3xl p-8 text-white text-center shadow-lg">
        <div className="text-5xl mb-3">👑</div>
        <h1 className="text-3xl font-bold mb-1">{tr("Learnthor Premium")}</h1>
        <p className="text-indigo-200 mb-6">{tr("Lifetime access · one-time payment")}</p>
        <p className="text-5xl font-bold mb-1">{formatIDR(99000)}</p>
        <p className="text-sm text-indigo-200">{tr('Lifetime access · one-time payment')}</p>
      </div>

      <div className="bg-white border border-gray-200 rounded-2xl mt-4 divide-y divide-gray-100 shadow-sm">
        {BENEFITS.map((b) => (
          <div key={tr(b.text)} className="flex items-center gap-3 px-5 py-3.5">
            <span className="text-xl">{b.icon}</span>
            <p className="text-sm text-gray-700">{b.text}</p>
          </div>
        ))}
      </div>

      {error && (
        <div className="mt-4 p-3 rounded-xl bg-red-50 border border-red-200 text-red-700 text-sm text-center">
          {typeof error === 'string' ? tr(error) : friendlyError(error, language)}
        </div>
      )}

      <button
        onClick={startCheckout}
        disabled={loading}
        className="w-full mt-6 py-4 rounded-2xl bg-indigo-600 text-white font-bold text-lg hover:bg-indigo-700 transition-colors disabled:opacity-60"
      >
        {loading ? tr("Preparing checkout…") : tr("Upgrade now →")}
      </button>

      <p className="text-center text-xs text-gray-400 mt-3">{tr("Secure payment via Midtrans · QRIS, e-wallets, bank transfer, cards")}</p>
    </div>
  );
}
