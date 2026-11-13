<script>
  import { toast } from '../core/toast.svelte.js';
</script>

{#if toast.msg}
  <div class="toast" role="status" style:--lift="{toast.lift}px">{toast.msg}</div>
{/if}

<style>
  .toast {
    position: fixed;
    z-index: 80;
    left: 50%;
    transform: translateX(-50%);
    bottom: calc(88px + var(--lift, 0px) + env(safe-area-inset-bottom));
    padding: 10px 18px;
    border-radius: 20px;
    background: var(--fg);
    color: var(--bg);
    font-size: 13px;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
    max-width: calc(100vw - 32px);
    animation: toast-in 0.2s ease-out;
  }
  @keyframes toast-in {
    from { opacity: 0; transform: translate(-50%, 10px); }
    to { opacity: 1; transform: translate(-50%, 0); }
  }
  /* Desktop: navigasi di samping, tidak ada bar bawah */
  @media (min-width: 900px) {
    .toast { bottom: calc(24px + var(--lift, 0px)); }
  }
</style>
