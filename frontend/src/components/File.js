// components/FileExplorer.js
import React, { useEffect, useState } from 'react';

const FileExplorer = ({ onLogout }) => {
  const [currentPath, setCurrentPath] = useState('/');
  const [content, setContent] = useState([]);
  const [fileContent, setFileContent] = useState('');
  const [loading, setLoading] = useState(false);

  const fetchDirectory = async (path) => {
    setLoading(true);
    setFileContent('');
    try {
      const response = await fetch('http://localhost:8080/api/command', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ input: `ls -path=${path}` }),
      });
      const data = await response.json();
      setContent(parseLsOutput(data.result));
    } catch (err) {
      setContent([]);
    } finally {
      setLoading(false);
    }
  };

  const parseLsOutput = (output) => {
    const lines = output.split('\n').filter(Boolean);
    return lines.map((line) => {
      const [type, name, permissions] = line.split(',');
      return {
        type: type.trim(),       // FILE or DIR
        name: name.trim(),
        permissions: permissions?.trim(),
      };
    });
  };

  const handleClick = (item) => {
    const newPath = currentPath.endsWith('/')
      ? currentPath + item.name
      : currentPath + '/' + item.name;

    if (item.type === 'DIR') {
      setCurrentPath(newPath);
      fetchDirectory(newPath);
    } else if (item.type === 'FILE') {
      fetchFile(newPath);
    }
  };

  const fetchFile = async (path) => {
    try {
      const response = await fetch('http://localhost:8080/api/command', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ input: `cat -path=${path}` }),
      });
      const data = await response.json();
      setFileContent(data.result);
    } catch (err) {
      setFileContent('Error al leer archivo');
    }
  };

  const handleGoBack = () => {
    if (currentPath === '/') return;
    const parts = currentPath.split('/');
    parts.pop();
    const newPath = parts.join('/') || '/';
    setCurrentPath(newPath);
    fetchDirectory(newPath);
  };

  useEffect(() => {
    fetchDirectory(currentPath);
  }, []);

  return (
    <div className="file-explorer">
      <div className="explorer-header">
        <h2>Sistema de Archivos</h2>
        <p><strong>Ruta actual:</strong> {currentPath}</p>
        <button onClick={handleGoBack}>⬅️ Atrás</button>
        <button onClick={onLogout}>Cerrar sesión</button>
      </div>

      {loading ? (
        <p>Cargando contenido...</p>
      ) : (
        <ul className="file-list">
          {content.map((item, index) => (
            <li key={index} onClick={() => handleClick(item)}>
              📁 {item.type === 'DIR' ? '📁' : '📄'} <strong>{item.name}</strong> — <code>{item.permissions}</code>
            </li>
          ))}
        </ul>
      )}

      {fileContent && (
        <div className="file-content">
          <h3>Contenido del archivo:</h3>
          <pre>{fileContent}</pre>
        </div>
      )}
    </div>
  );
};

export default FileExplorer;
