<script lang="ts">
  import { svgAssetMarkup } from './customSvg';

  interface Props {
    src: string;
    label?: string;
    classes?: string;
  }

  let { src, label = '', classes = 'h-4 w-4' }: Props = $props();
  let markup = $derived(svgAssetMarkup(src, label, ['inline-block shrink-0', classes].filter(Boolean).join(' ')));
</script>

<span class="inline-flex shrink-0 text-current [&>svg]:fill-current" aria-hidden={label ? undefined : 'true'}>
  {@html markup}
</span>
