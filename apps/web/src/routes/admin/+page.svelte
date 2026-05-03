<script>
	import { onMount } from 'svelte';

	// view: 'login' | 'change-password' | 'dashboard'
	let view = 'login';

	// Login form
	let username = '';
	let password = '';
	let loginError = '';
	let loginBusy = false;

	// Change-password form
	let cpCurrent = '';
	let cpNew = '';
	let cpConfirm = '';
	let cpError = '';
	let cpBusy = false;

	// Dashboard
	let devices = [];
	let devicesError = '';
	// rename state: { [mac]: { editing: bool, value: string, busy: bool, error: string } }
	let renameState = {};

	onMount(async () => {
		// Try a privileged endpoint to detect an existing valid session.
		// If the backend returns 401 we stay on the login view.
		const res = await fetch('/api/devices');
		if (res.ok) {
			// /api/devices is public so this just warms the connection.
			// To verify an admin session we call change-password with empty body
			// to provoke a 401 without side-effects. But that is noisy.
			// Instead: stay on login until the user actively logs in.
			// The only exception is if we stored a view hint in sessionStorage
			// (set after a successful login, cleared on logout).
			if (sessionStorage.getItem('ph_admin') === '1') {
				await loadDashboard();
			}
		}
	});

	async function handleLogin(e) {
		e.preventDefault();
		loginError = '';
		loginBusy = true;
		try {
			const res = await fetch('/api/auth/login', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ username, password })
			});
			if (res.status === 429) {
				loginError = 'För många försök. Vänta en stund och försök igen.';
				return;
			}
			if (!res.ok) {
				loginError = 'Fel användarnamn eller lösenord.';
				return;
			}
			const data = await res.json();
			sessionStorage.setItem('ph_admin', '1');
			if (data.must_change_password) {
				view = 'change-password';
			} else {
				await loadDashboard();
			}
		} catch {
			loginError = 'Nätverksfel. Kontrollera anslutningen.';
		} finally {
			loginBusy = false;
		}
	}

	async function handleChangePassword(e) {
		e.preventDefault();
		cpError = '';
		if (cpNew !== cpConfirm) {
			cpError = 'De nya lösenorden matchar inte.';
			return;
		}
		if (cpNew.length < 12) {
			cpError = 'Nytt lösenord måste vara minst 12 tecken.';
			return;
		}
		cpBusy = true;
		try {
			const res = await fetch('/api/auth/change-password', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ current: cpCurrent, new: cpNew })
			});
			if (res.status === 401) {
				cpError = 'Nuvarande lösenord stämmer inte.';
				return;
			}
			if (!res.ok) {
				cpError = 'Något gick fel. Försök igen.';
				return;
			}
			cpCurrent = '';
			cpNew = '';
			cpConfirm = '';
			await loadDashboard();
		} catch {
			cpError = 'Nätverksfel.';
		} finally {
			cpBusy = false;
		}
	}

	async function loadDashboard() {
		devicesError = '';
		const res = await fetch('/api/devices');
		if (res.status === 401) {
			sessionStorage.removeItem('ph_admin');
			view = 'login';
			return;
		}
		if (!res.ok) {
			devicesError = 'Kunde inte hämta enhetslista.';
			view = 'dashboard';
			return;
		}
		const list = await res.json();
		devices = list;
		renameState = {};
		for (const d of list) {
			renameState[d.mac] = {
				editing: false,
				value: d.display_name || d.hostname || '',
				busy: false,
				error: ''
			};
		}
		view = 'dashboard';
	}

	async function handleLogout() {
		await fetch('/api/auth/logout', { method: 'POST' });
		sessionStorage.removeItem('ph_admin');
		username = '';
		password = '';
		view = 'login';
	}

	function startEdit(mac) {
		renameState[mac] = { ...renameState[mac], editing: true, error: '' };
		renameState = renameState;
	}

	function cancelEdit(mac) {
		const d = devices.find((x) => x.mac === mac);
		renameState[mac] = {
			...renameState[mac],
			editing: false,
			value: d?.display_name || d?.hostname || '',
			error: ''
		};
		renameState = renameState;
	}

	async function saveRename(mac) {
		const val = renameState[mac]?.value?.trim();
		if (!val) {
			renameState[mac] = { ...renameState[mac], error: 'Namnet får inte vara tomt.' };
			renameState = renameState;
			return;
		}
		renameState[mac] = { ...renameState[mac], busy: true, error: '' };
		renameState = renameState;
		try {
			const res = await fetch(`/api/devices/${encodeURIComponent(mac)}/rename`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ display_name: val })
			});
			if (res.status === 401) {
				sessionStorage.removeItem('ph_admin');
				view = 'login';
				return;
			}
			if (!res.ok) {
				renameState[mac] = { ...renameState[mac], error: 'Kunde inte spara.' };
				return;
			}
			devices = devices.map((d) =>
				d.mac === mac ? { ...d, display_name: val } : d
			);
			renameState[mac] = { ...renameState[mac], editing: false };
		} finally {
			renameState[mac] = { ...renameState[mac], busy: false };
			renameState = renameState;
		}
	}

	function deviceLabel(d) {
		return d.display_name || d.hostname || d.mac;
	}
</script>

<main>
	<a href="/" class="back">&larr; Tillbaka</a>

	{#if view === 'login'}
		<h1>Admin</h1>
		<form class="form" on:submit={handleLogin}>
			<label for="un">Användarnamn</label>
			<input
				id="un"
				type="text"
				bind:value={username}
				autocomplete="username"
				required
				placeholder="admin"
			/>

			<label for="pw">Lösenord</label>
			<input
				id="pw"
				type="password"
				bind:value={password}
				autocomplete="current-password"
				required
				placeholder="Lösenord"
			/>

			{#if loginError}
				<p class="form-error" role="alert">{loginError}</p>
			{/if}

			<button type="submit" class="btn-primary" disabled={loginBusy}>
				{loginBusy ? 'Loggar in...' : 'Logga in'}
			</button>
		</form>

	{:else if view === 'change-password'}
		<h1>Byt lösenord</h1>
		<p class="info-msg">Du måste byta lösenord innan du kan fortsätta.</p>
		<form class="form" on:submit={handleChangePassword}>
			<label for="cp-cur">Nuvarande lösenord</label>
			<input
				id="cp-cur"
				type="password"
				bind:value={cpCurrent}
				autocomplete="current-password"
				required
			/>

			<label for="cp-new">Nytt lösenord (minst 12 tecken)</label>
			<input
				id="cp-new"
				type="password"
				bind:value={cpNew}
				autocomplete="new-password"
				required
				minlength="12"
			/>

			<label for="cp-confirm">Bekräfta nytt lösenord</label>
			<input
				id="cp-confirm"
				type="password"
				bind:value={cpConfirm}
				autocomplete="new-password"
				required
			/>

			{#if cpError}
				<p class="form-error" role="alert">{cpError}</p>
			{/if}

			<button type="submit" class="btn-primary" disabled={cpBusy}>
				{cpBusy ? 'Sparar...' : 'Spara nytt lösenord'}
			</button>
		</form>

	{:else if view === 'dashboard'}
		<div class="dash-header">
			<h1>Enhetshantering</h1>
			<button class="btn-logout" on:click={handleLogout}>Logga ut</button>
		</div>

		{#if devicesError}
			<p class="form-error" role="alert">{devicesError}</p>
		{/if}

		{#if devices.length === 0 && !devicesError}
			<p class="muted">Inga enheter i databasen ännu.</p>
		{/if}

		<ul class="device-list">
			{#each devices as d (d.mac)}
				{@const rs = renameState[d.mac] ?? { editing: false, value: '', busy: false, error: '' }}
				<li class="device-item">
					<div class="device-meta">
						<span class="device-label">{deviceLabel(d)}</span>
						<span class="device-sub">{d.ip} &middot; {d.mac}</span>
					</div>

					{#if rs.editing}
						<div class="rename-row">
							<input
								class="rename-input"
								type="text"
								bind:value={renameState[d.mac].value}
								placeholder="Visningsnamn"
								maxlength="80"
							/>
							<button
								class="btn-save"
								disabled={rs.busy}
								on:click={() => saveRename(d.mac)}
							>
								{rs.busy ? '...' : 'Spara'}
							</button>
							<button class="btn-cancel" on:click={() => cancelEdit(d.mac)}>Avbryt</button>
						</div>
						{#if rs.error}
							<p class="inline-error" role="alert">{rs.error}</p>
						{/if}
					{:else}
						<button class="btn-rename" on:click={() => startEdit(d.mac)}>Byt namn</button>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
</main>

<style>
	:global(body) {
		font-family: system-ui, sans-serif;
		background: #1a0a0a;
		color: #e0e0e0;
		min-height: 100vh;
		display: flex;
		justify-content: center;
		align-items: flex-start;
	}

	main {
		max-width: 500px;
		width: 100%;
		padding: 2.5rem 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}

	.back {
		display: inline-block;
		color: #f7b8b8;
		text-decoration: none;
		font-size: 1rem;
	}

	.back:hover {
		text-decoration: underline;
	}

	h1 {
		font-size: 2rem;
		color: #e74c3c;
	}

	.info-msg {
		color: #ffe5a0;
		background: #3a2a00;
		padding: 0.75rem 1rem;
		border-radius: 0.5rem;
		font-size: 0.95rem;
	}

	.form {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	label {
		font-size: 0.85rem;
		color: #aaa;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	input[type='text'],
	input[type='password'] {
		width: 100%;
		padding: 0.85rem 1rem;
		font-size: 1rem;
		background: #2a1010;
		border: 1px solid #5a2020;
		border-radius: 0.5rem;
		color: #e0e0e0;
		outline: none;
	}

	input[type='text']:focus,
	input[type='password']:focus {
		border-color: #e74c3c;
	}

	.form-error {
		color: #e74c3c;
		font-size: 0.9rem;
		margin: 0;
	}

	.btn-primary {
		width: 100%;
		padding: 0.9rem;
		font-size: 1.05rem;
		font-weight: bold;
		background: #e74c3c;
		color: #fff;
		border: none;
		border-radius: 0.5rem;
		cursor: pointer;
		transition: filter 0.15s;
		margin-top: 0.25rem;
	}

	.btn-primary:hover:not(:disabled) {
		filter: brightness(1.1);
	}

	.btn-primary:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.dash-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
	}

	.btn-logout {
		background: #2a1010;
		color: #f5a0a0;
		border: 1px solid #5a2020;
		border-radius: 0.4rem;
		padding: 0.5rem 1rem;
		font-size: 0.9rem;
		cursor: pointer;
		transition: background 0.15s;
		white-space: nowrap;
	}

	.btn-logout:hover {
		background: #3a1a1a;
	}

	.device-list {
		list-style: none;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.device-item {
		background: #2a1010;
		border-radius: 0.5rem;
		padding: 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.device-meta {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}

	.device-label {
		font-weight: bold;
		font-size: 1rem;
	}

	.device-sub {
		font-size: 0.78rem;
		color: #888;
		font-family: monospace;
	}

	.rename-row {
		display: flex;
		gap: 0.4rem;
		flex-wrap: wrap;
	}

	.rename-input {
		flex: 1;
		min-width: 120px;
		padding: 0.45rem 0.7rem;
		font-size: 0.95rem;
		background: #1a0a0a;
		border: 1px solid #5a2020;
		border-radius: 0.4rem;
		color: #e0e0e0;
		outline: none;
	}

	.rename-input:focus {
		border-color: #e74c3c;
	}

	.btn-save {
		background: #2d6a4f;
		color: #fff;
		border: none;
		border-radius: 0.4rem;
		padding: 0.45rem 0.9rem;
		font-size: 0.9rem;
		cursor: pointer;
		transition: filter 0.15s;
	}

	.btn-save:hover:not(:disabled) {
		filter: brightness(1.15);
	}

	.btn-save:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.btn-cancel {
		background: transparent;
		color: #aaa;
		border: 1px solid #444;
		border-radius: 0.4rem;
		padding: 0.45rem 0.9rem;
		font-size: 0.9rem;
		cursor: pointer;
	}

	.btn-cancel:hover {
		color: #e0e0e0;
		border-color: #888;
	}

	.btn-rename {
		align-self: flex-start;
		background: transparent;
		color: #7eb8f7;
		border: 1px solid #2a4a7a;
		border-radius: 0.4rem;
		padding: 0.35rem 0.75rem;
		font-size: 0.85rem;
		cursor: pointer;
		transition: background 0.15s;
	}

	.btn-rename:hover {
		background: #1a2a4a;
	}

	.inline-error {
		color: #e74c3c;
		font-size: 0.85rem;
		margin: 0;
	}

	.muted {
		color: #666;
	}

	@media (max-width: 375px) {
		main {
			padding: 1.5rem 1rem;
		}

		h1 {
			font-size: 1.6rem;
		}

		.dash-header {
			flex-direction: column;
			align-items: flex-start;
		}

		.rename-row {
			flex-direction: column;
		}

		.rename-input {
			min-width: unset;
			width: 100%;
		}
	}
</style>
