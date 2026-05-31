import Content from "./components/Content";
import Header from "./components/Header";
import Sidebar from "./components/Sidebar";

const Preference = () => {
  return (
    <div className="flex h-screen gap-2" data-tauri-drag-region>
      <Sidebar />

      <div className="flex h-full flex-1 flex-col" data-tauri-drag-region>
        <Header />
        <Content />
      </div>
    </div>
  );
};

export default Preference;
