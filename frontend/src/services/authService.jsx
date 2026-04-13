export const loginWithYandex = async () => {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          accessToken: "fake-token",
          user: {
            name: "Polina",
          },
        });
      }, 1000);
    });
  };