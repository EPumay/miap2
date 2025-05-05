import React, { useCallback, useEffect, useState } from 'react';

const FileExplorer = ({ onLogout }) => {
  const [currentPath, setCurrentPath] = useState('/');
  const [entries, setEntries] = useState([]);

  const fetchDirectory = useCallback(async (path) => {
    const res = await fetch('http://localhost:8080/api/command', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ input: `ls -path=${path}` }),
    });
    const data = await res.json();
    const parsed = data.result
      .trim()
      .split('\n')
      .map((line) => {
        const [type, name, perms] = line.split(',');
        return { type, name, perms };
      });
    setEntries(parsed);
  }, []);

  useEffect(() => {
    fetchDirectory(currentPath);
  }, [currentPath, fetchDirectory]);

  return (
    <div>
      <h2>Explorador de Archivos - {currentPath}</h2>
      <button onClick={onLogout}>Cerrar Sesión</button>
      <ul>
        {entries.map((e, idx) => (
          <li key={idx}>
            {e.type === 'DIR' ? (
              <button onClick={() => setCurrentPath(`${currentPath}${e.name}/`)}>
                📁 {e.name} ({e.perms})
              </button>
            ) : (
              <span>📄 {e.name} ({e.perms})</span>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
};

export default FileExplorer;
