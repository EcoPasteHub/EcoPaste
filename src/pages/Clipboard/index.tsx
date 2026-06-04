import Footer from "./components/Footer";
import Group from "./components/Group";
import Header from "./components/Header";
import List from "./components/List";

const Clipboard = () => {
  return (
    <div
      className="flex size-screen flex-col overflow-hidden bg-container"
      data-tauri-drag-region
    >
      <Header />

      <Group />

      <List />

      <Footer />
    </div>
  );
};

export default Clipboard;
