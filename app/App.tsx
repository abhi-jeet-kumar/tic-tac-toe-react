import React from 'react';
import { Text, Button, SafeAreaView } from 'react-native';
import { AuthProvider, useAuth } from './src/providers/AuthProvider';

function Home() {
  const { isAuthed, login, logout, username } = useAuth();
  return (
    <SafeAreaView style={{ flex: 1, alignItems: 'center', justifyContent: 'center' }}>
      {!isAuthed ? (
        <>
          <Text style={{ marginBottom: 12 }}>Welcome to TicTacToe</Text>
          <Button title="Continue" onPress={() => login()} />
        </>
      ) : (
        <>
          <Text style={{ marginBottom: 12 }}>Signed in as {username}</Text>
          <Button title="Logout" onPress={logout} />
        </>
      )}
    </SafeAreaView>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <Home />
    </AuthProvider>
  );
}

