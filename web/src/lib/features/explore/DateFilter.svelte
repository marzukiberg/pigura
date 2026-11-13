<script>
  import { Popover, Calendar, RangeCalendar } from 'bits-ui';
  import { CalendarDate, getLocalTimeZone, today } from '@internationalized/date';
  import Icon from '../../ui/Icon.svelte';
  import Seg from '../../ui/Seg.svelte';

  // onchange({ from, to } | null): detik unix, from inklusif, to eksklusif.
  let { onchange = () => {} } = $props();

  const tz = getLocalTimeZone();
  const now = today(tz);
  const LOCALE = 'id-ID';
  const MODES = [
    { id: 'day', label: 'Tanggal' },
    { id: 'month', label: 'Bulan' },
    { id: 'range', label: 'Rentang' },
  ];
  const MONTHS = Array.from({ length: 12 }, (_, i) =>
    new Intl.DateTimeFormat(LOCALE, { month: 'short' }).format(new Date(2000, i, 1))
  );

  let open = $state(false);
  let mode = $state('day');
  let day = $state(undefined);
  let range = $state({ start: undefined, end: undefined });
  let month = $state(null); // { y, m } m = 1..12
  let year = $state(now.year);

  const sec = (d) => Math.floor(d.toDate(tz).getTime() / 1000);
  const fmt = (d, o) => new Intl.DateTimeFormat(LOCALE, o).format(d.toDate(tz));

  let label = $derived.by(() => {
    if (mode === 'day' && day) return fmt(day, { day: 'numeric', month: 'long', year: 'numeric' });
    if (mode === 'month' && month)
      return fmt(new CalendarDate(month.y, month.m, 1), { month: 'long', year: 'numeric' });
    if (mode === 'range' && (range.start || range.end)) {
      const f = (d) => fmt(d, { day: 'numeric', month: 'short', year: 'numeric' });
      return `${range.start ? f(range.start) : '…'} – ${range.end ? f(range.end) : '…'}`;
    }
    return '';
  });

  function emit() {
    let r = null;
    if (mode === 'day' && day) r = { from: sec(day), to: sec(day.add({ days: 1 })) };
    else if (mode === 'month' && month) {
      const s = new CalendarDate(month.y, month.m, 1);
      r = { from: sec(s), to: sec(s.add({ months: 1 })) };
    } else if (mode === 'range' && range.start) {
      r = { from: sec(range.start), to: range.end ? sec(range.end.add({ days: 1 })) : undefined };
    }
    onchange(r);
  }

  function setMode(m) {
    if (m === mode) return;
    mode = m;
    day = undefined;
    range = { start: undefined, end: undefined };
    month = null;
    emit();
  }

  function clear() {
    day = undefined;
    range = { start: undefined, end: undefined };
    month = null;
    emit();
  }

  function pickMonth(m) {
    month = { y: year, m };
    emit();
    open = false;
  }

  function preset(days) {
    range = { start: now.subtract({ days: days - 1 }), end: now };
    emit();
    open = false;
  }
  function presetThisMonth() {
    range = { start: new CalendarDate(now.year, now.month, 1), end: now };
    emit();
    open = false;
  }
</script>

<div class="df">
  <Popover.Root bind:open>
    <Popover.Trigger class="df-trigger {label ? 'has-value' : ''}">
      <Icon name="calendar" size={16} />
      <span>{label || 'Pilih tanggal'}</span>
    </Popover.Trigger>
    {#if label}
      <button class="df-clear" onclick={clear} aria-label="Hapus filter tanggal" title="Hapus filter">
        <Icon name="close" size={16} />
      </button>
    {/if}

    <Popover.Portal>
      <Popover.Content class="df-pop" sideOffset={8} align="start" collisionPadding={12}>
        <Seg class="df-modes" role="tab" size="xs" label="Jenis filter tanggal" options={MODES} value={mode} onchange={setMode} />

        {#if mode === 'day'}
          <Calendar.Root
            type="single"
            bind:value={day}
            onValueChange={() => {
              emit();
              open = false;
            }}
            locale={LOCALE}
            weekStartsOn={1}
            weekdayFormat="short"
            fixedWeeks={true}
            maxValue={now}
            preventDeselect
            class="df-cal"
          >
            {#snippet children({ months, weekdays })}
              <Calendar.Header class="df-head">
                <Calendar.PrevButton class="df-nav" aria-label="Bulan sebelumnya">
                  <Icon name="left" size={18} />
                </Calendar.PrevButton>
                <Calendar.Heading class="df-heading" />
                <Calendar.NextButton class="df-nav" aria-label="Bulan berikutnya">
                  <Icon name="right" size={18} />
                </Calendar.NextButton>
              </Calendar.Header>
              {#each months as mo (mo.value.toString())}
                <Calendar.Grid class="df-grid">
                  <Calendar.GridHead>
                    <Calendar.GridRow class="df-row">
                      {#each weekdays as wd, i (i)}
                        <Calendar.HeadCell class="df-hcell">{wd}</Calendar.HeadCell>
                      {/each}
                    </Calendar.GridRow>
                  </Calendar.GridHead>
                  <Calendar.GridBody>
                    {#each mo.weeks as week, wi (wi)}
                      <Calendar.GridRow class="df-row">
                        {#each week as date (date.toString())}
                          <Calendar.Cell {date} month={mo.value} class="df-cell">
                            <Calendar.Day class="df-day" />
                          </Calendar.Cell>
                        {/each}
                      </Calendar.GridRow>
                    {/each}
                  </Calendar.GridBody>
                </Calendar.Grid>
              {/each}
            {/snippet}
          </Calendar.Root>
        {:else if mode === 'range'}
          <RangeCalendar.Root
            bind:value={range}
            onValueChange={(v) => {
              range = v;
              emit();
              if (v.start && v.end) open = false;
            }}
            locale={LOCALE}
            weekStartsOn={1}
            weekdayFormat="short"
            fixedWeeks={true}
            maxValue={now}
            class="df-cal"
          >
            {#snippet children({ months, weekdays })}
              <RangeCalendar.Header class="df-head">
                <RangeCalendar.PrevButton class="df-nav" aria-label="Bulan sebelumnya">
                  <Icon name="left" size={18} />
                </RangeCalendar.PrevButton>
                <RangeCalendar.Heading class="df-heading" />
                <RangeCalendar.NextButton class="df-nav" aria-label="Bulan berikutnya">
                  <Icon name="right" size={18} />
                </RangeCalendar.NextButton>
              </RangeCalendar.Header>
              {#each months as mo (mo.value.toString())}
                <RangeCalendar.Grid class="df-grid">
                  <RangeCalendar.GridHead>
                    <RangeCalendar.GridRow class="df-row">
                      {#each weekdays as wd, i (i)}
                        <RangeCalendar.HeadCell class="df-hcell">{wd}</RangeCalendar.HeadCell>
                      {/each}
                    </RangeCalendar.GridRow>
                  </RangeCalendar.GridHead>
                  <RangeCalendar.GridBody>
                    {#each mo.weeks as week, wi (wi)}
                      <RangeCalendar.GridRow class="df-row">
                        {#each week as date (date.toString())}
                          <RangeCalendar.Cell {date} month={mo.value} class="df-cell range">
                            <RangeCalendar.Day class="df-day" />
                          </RangeCalendar.Cell>
                        {/each}
                      </RangeCalendar.GridRow>
                    {/each}
                  </RangeCalendar.GridBody>
                </RangeCalendar.Grid>
              {/each}
            {/snippet}
          </RangeCalendar.Root>
          <div class="df-presets">
            <button onclick={() => preset(7)}>7 hari</button>
            <button onclick={() => preset(30)}>30 hari</button>
            <button onclick={presetThisMonth}>Bulan ini</button>
          </div>
        {:else}
          <div class="df-head">
            <button class="df-nav" onclick={() => year--} aria-label="Tahun sebelumnya">
              <Icon name="left" size={18} />
            </button>
            <div class="df-heading">{year}</div>
            <button
              class="df-nav"
              onclick={() => year++}
              disabled={year >= now.year}
              aria-label="Tahun berikutnya"
            >
              <Icon name="right" size={18} />
            </button>
          </div>
          <div class="df-months">
            {#each MONTHS as name, i (i)}
              {@const future = year === now.year && i + 1 > now.month}
              <button
                class="df-month"
                class:selected={month && month.y === year && month.m === i + 1}
                disabled={future}
                onclick={() => pickMonth(i + 1)}
              >
                {name}
              </button>
            {/each}
          </div>
        {/if}
      </Popover.Content>
    </Popover.Portal>
  </Popover.Root>
</div>

<style>
  .df {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    max-width: 100%;
    min-width: 0;
  }

  :global(.df-trigger > span) {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }

  :global(.df-trigger) {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    height: 36px;
    padding: 0 16px;
    border-radius: 18px;
    background: var(--surface-variant);
    color: var(--fg);
    font-size: 13px;
    font-weight: 500;
    border: 1px solid var(--line-subtle);
    cursor: pointer;
    transition: background 0.15s, border-color 0.15s;
  }
  :global(.df-trigger:hover) { background: var(--hover); }
  :global(.df-trigger.has-value),
  :global(.df-trigger[data-state='open']) {
    background: var(--acc-bg);
    color: var(--acc-fg);
    border-color: var(--acc);
  }

  .df-clear {
    display: inline-grid;
    place-items: center;
    width: 32px;
    height: 32px;
    border-radius: 50%;
    color: var(--mut);
  }
  .df-clear:hover { background: var(--hover); color: var(--fg); }

  :global(.df-pop) {
    z-index: 60;
    width: min(320px, calc(100vw - 24px));
    padding: 12px;
    border-radius: 16px;
    background: var(--surface);
    color: var(--fg);
    border: 1px solid var(--line);
    box-shadow: 0 8px 28px rgba(0, 0, 0, 0.22);
    outline: none;
  }

  :global(.df-modes) { margin-bottom: 10px; }

  :global(.df-head) {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 6px;
  }
  :global(.df-heading) {
    font-size: 14px;
    font-weight: 600;
    text-transform: capitalize;
  }
  :global(.df-nav) {
    display: inline-grid;
    place-items: center;
    width: 36px;
    height: 36px;
    border-radius: 50%;
    color: var(--fg);
    cursor: pointer;
  }
  :global(.df-nav:hover:not(:disabled)) { background: var(--hover); }
  :global(.df-nav:disabled),
  :global(.df-nav[data-disabled]) { opacity: 0.3; cursor: default; }

  :global(.df-grid) { width: 100%; border-collapse: collapse; user-select: none; }
  :global(.df-row) { display: flex; width: 100%; }
  :global(.df-hcell) {
    flex: 1;
    height: 28px;
    font-size: 11px;
    font-weight: 500;
    color: var(--mut);
    text-align: center;
    text-transform: capitalize;
  }
  :global(.df-cell) {
    flex: 1;
    padding: 1px 0;
    text-align: center;
  }
  :global(.df-day) {
    position: relative;
    display: inline-grid;
    place-items: center;
    width: 100%;
    max-width: 40px;
    height: 36px;
    border-radius: 50%;
    font-size: 13px;
    cursor: pointer;
    color: var(--fg);
  }
  :global(.df-day:hover) { background: var(--hover); }
  :global(.df-day[data-today]) { box-shadow: inset 0 0 0 1px var(--acc); }
  :global(.df-day[data-outside-month]) { visibility: hidden; pointer-events: none; }
  :global(.df-day[data-disabled]) { color: var(--mut); opacity: 0.35; pointer-events: none; }
  :global(.df-day[data-selected]) {
    background: var(--acc);
    color: #fff;
    font-weight: 600;
  }
  :global(.df-day[data-selected]:hover) { background: var(--acc-hover); }
  /* rentang: hari di tengah berupa pita, ujung berupa lingkaran penuh */
  :global(.df-cell.range:has([data-selection-start]):not(:has([data-selection-end]))),
  :global(.df-cell.range:has([data-selection-start][data-selection-end])) { background: none; }
  :global(.df-cell.range .df-day[data-selected]:not([data-selection-start]):not([data-selection-end])) {
    background: var(--acc-bg);
    color: var(--acc-fg);
    border-radius: 0;
    max-width: none;
  }
  :global(.df-cell.range:has([data-selection-start]):not(:has([data-selection-end]))) { background: none; }
  :global(.df-cell.range .df-day[data-selection-start]) { border-radius: 50%; }

  :global(.df-presets) {
    display: flex;
    gap: 6px;
    margin-top: 10px;
    padding-top: 10px;
    border-top: 1px solid var(--line);
  }
  :global(.df-presets button) {
    flex: 1;
    height: 30px;
    border-radius: 15px;
    font-size: 12px;
    font-weight: 500;
    border: 1px solid var(--line);
    color: var(--fg);
    cursor: pointer;
  }
  :global(.df-presets button:hover) { background: var(--hover); }

  :global(.df-months) {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 6px;
    margin-top: 4px;
  }
  :global(.df-month) {
    height: 44px;
    border-radius: 12px;
    font-size: 13px;
    font-weight: 500;
    text-transform: capitalize;
    color: var(--fg);
    cursor: pointer;
  }
  :global(.df-month:hover:not(:disabled)) { background: var(--hover); }
  :global(.df-month:disabled) { opacity: 0.3; cursor: default; }
  :global(.df-month.selected) { background: var(--acc); color: #fff; }

  @media (pointer: coarse) {
    :global(.df-day) { height: 40px; }
    :global(.df-month) { height: 48px; }
  }
</style>
