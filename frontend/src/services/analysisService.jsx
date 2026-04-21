import { getAccessToken } from "./authService";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "";

export async function uploadImages(files, brand) {
    console.log("MOCK UPLOAD:", files, brand);
  
    await new Promise((resolve) => setTimeout(resolve, 1500));
  
    return {
      analysis_id: "mock-123",
    };
}

export async function getAnalysis(id) {
    console.log("GET ANALYSIS:", id);
  
    await new Promise((resolve) => setTimeout(resolve, 2000));
  
    return {
      status: "done",
      image_url: "https://via.placeholder.com/400",
      damages: [
        { part: "Бампер", severity: "Средний" },
        { part: "Крыло", severity: "Лёгкий" },
      ],
      services: [
        {
          name: "AutoFix Riga",
          phone: "+371 12345678",
          address: "Riga, LV",
          description: "Кузовной ремонт любой сложности",
        },
        {
          name: "CarRepair Pro",
          phone: "+371 87654321",
          address: "Riga, LV",
        },
      ],
    };
}

export async function getAnalysisHistory() {
    await new Promise((resolve) => setTimeout(resolve, 1000));
  
    return [
      {
        id: "1",
        created_at: "2026-04-20",
        brand: "BMW",
      },
      {
        id: "2",
        created_at: "2026-04-19",
        brand: "Audi",
      },
      {
        id: "3",
        created_at: "2026-04-18",
        brand: "Toyota",
      },
    ];
}