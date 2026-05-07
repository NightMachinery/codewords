<script lang="ts">
  interface Props {
    src: string;
    label?: string;
    classes?: string;
  }

  let { src, label = '', classes = 'h-4 w-4' }: Props = $props();
</script>

<span
  class={['inline-block shrink-0 bg-current', classes].filter(Boolean).join(' ')}
  style={`-webkit-mask: url(${src}) center / contain no-repeat; mask: url(${src}) center / contain no-repeat;`}
  aria-label={label || undefined}
  aria-hidden={label ? undefined : 'true'}
></span>
