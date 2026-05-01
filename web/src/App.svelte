<script lang="ts">
  import { BrowserRouter, Route, Routes } from '@hvniel/svelte-router';

  import HomePage from './pages/HomePage.svelte';
  import NotFoundPage from './pages/NotFoundPage.svelte';
  import RoomPage from './pages/RoomPage.svelte';
</script>

<BrowserRouter>
  <Routes>
    <Route path="/" Component={HomePage} />
    <Route path="/rooms/:roomId" Component={RoomPage} />
    <Route path="/room/:roomId" Component={RoomPage} />
    <Route path="*" Component={NotFoundPage} />
  </Routes>
</BrowserRouter>
